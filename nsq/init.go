package nsq

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/garyburd/redigo/redis"
	logging "gopkg.in/tokopedia/logging.v1"
)

type ServerConfig struct {
	Name string
}

type Config struct {
	Server ServerConfig
	NSQ    NSQConfig
	Redis  RedisConfig
}
type RedisConfig struct {
	Connection string
}

type NSQConfig struct {
	NSQD     string
	Lookupds string
}

type NSQModule struct {
	cfg   *Config
	q     *nsq.Consumer
	Redis *redis.Pool
}

var NSQHelper *NSQModule

func NewNSQModule() *NSQModule {

	var cfg Config

	ok := logging.ReadModuleConfig(&cfg, "config", "nsq") || logging.ReadModuleConfig(&cfg, "../../files/etc/gosample", "nsq")
	if !ok {
		// when the app is run with -e switch, this message will automatically be redirected to the log file specified
		log.Fatalln("failed to read nsq config")
	}

	// this message only shows up if app is run with -debug option, so its great for debugging
	logging.Debug.Println("nsq init called", cfg.Server.Name)

	redisPools := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", cfg.Redis.Connection)
			if err != nil {
				return nil, err
			}
			return conn, err
		},
	}

	NSQHelper = &NSQModule{
		cfg:   &cfg,
		Redis: redisPools,
	}
	// contohnya: caranya ciptakan nsq consumer
	nsqCfg := nsq.NewConfig()
	q := createNewConsumer(nsqCfg, "Counter_Visitor_NSQ", "counter_visitor_channel", NSQHelper.PrintMessage)
	q.SetLogger(log.New(os.Stderr, "nsq:", log.Ltime), nsq.LogLevelError)
	q.ConnectToNSQLookupd("devel-go.tkpd:4161")

	NSQHelper.q = q

	return NSQHelper

}
func createNewConsumer(nsqCfg *nsq.Config, topic string, channel string, handler nsq.HandlerFunc) *nsq.Consumer {
	q, err := nsq.NewConsumer(topic, channel, nsqCfg)
	if err != nil {
		log.Fatal("failed to create consumer for ", topic, channel, err)
	}
	q.AddHandler(handler)
	return q
}

type Message struct {
	Counter int64 `json:"counter"`
}

func (nsq *NSQModule) PrintMessage(msg *nsq.Message) error {
	log.Println("got message :", msg.Body)

	var message Message
	err := json.Unmarshal(msg.Body, &message)
	if err != nil {
		fmt.Println("There was an error:", err)
	}

	pool := nsq.Redis.Get()
	_, err_set := redis.String(pool.Do("SET", "counter", message.Counter))
	if err_set != nil {
		log.Printf("Failed Error: %s\n", err_set.Error())
	}
	msg.Finish()
	return nil
}
