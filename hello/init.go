package hello

import (
	"encoding/json"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"text/template"

	nsqq "hello/nsq"

	nsq "github.com/nsqio/go-nsq"

	"github.com/garyburd/redigo/redis"
	_ "github.com/lib/pq"
	"github.com/tokopedia/sqlt"
	logging "gopkg.in/tokopedia/logging.v1"
)

type ServerConfig struct {
	Name string
}

type DatabaseConfig struct {
	Type       string
	Connection string
}

type NSQConfig struct {
	NSQD     string
	Lookupds string
}

type RedisConfig struct {
	Connection string
}

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	NSQ      NSQConfig
}

type HelloWorldModule struct {
	cfg       *Config
	DB        *sqlt.DB
	NSQ       *nsq.Producer
	something string
	stats     *expvar.Int
	render    *template.Template //FOR TRAINING
	Redis     *redis.Pool
}

func NewHelloWorldModule() *HelloWorldModule {

	var cfg Config

	ok := logging.ReadModuleConfig(&cfg, "config", "hello") || logging.ReadModuleConfig(&cfg, "files/etc/gosample", "hello")
	if !ok {
		// when the app is run with -e switch, this message will automatically be redirected to the log file specified
		log.Fatalln("failed to read config")
	}

	masterDB := cfg.Database.Connection
	slaveDB := cfg.Database.Connection
	dbConnection := fmt.Sprintf("%s;%s", masterDB, slaveDB)

	db, err := sqlt.Open(cfg.Database.Type, dbConnection)
	if err != nil {
		log.Fatalln("Failed to connect database. Error: ", err.Error())
	}

	redisPools := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", cfg.Redis.Connection)
			if err != nil {
				return nil, err
			}
			return conn, err
		},
	}

	producer, err := nsq.NewProducer(cfg.NSQ.NSQD, nsq.NewConfig())
	if err != nil {
		log.Fatalln("failed to create nsq producer. Error: ", err.Error())
	}

	// this message only shows up if app is run with -debug option, so its great for debugging
	logging.Debug.Println("hello init called", cfg.Server.Name)

	//FOR TRAINING
	engine := template.Must(template.ParseGlob("files/var/templates/*"))

	return &HelloWorldModule{
		cfg:       &cfg,
		DB:        db,
		NSQ:       producer,
		Redis:     redisPools,
		something: "John Doe",
		stats:     expvar.NewInt("rpsStats"),
		render:    engine, //FOR TRAINING
	}

}

type User struct {
	User_id     int64  `db:"user_id"`
	Full_name   string `db:"full_name"`
	User_email  string `db:"user_email"`
	Msisdn      string `db:"msisdn"`
	Age         string
	Update_time string `db:"update_time"`
}

func (hlm *HelloWorldModule) GetMultiDataFromDatabase(w http.ResponseWriter, r *http.Request) {
	hlm.stats.Add(1)
	name := r.FormValue("name")
	res := []User{}
	query := "SELECT user_id, full_name, user_email, msisdn, COALESCE(Age(birth_date), '0 years') as age, COALESCE(update_time, CURRENT_TIMESTAMP) as update_time FROM ws_user WHERE full_name ILIKE $1  LIMIT 10"
	err := hlm.DB.Select(&res, query, name+"%")
	if err != nil {
		log.Println("Error Query Database. Error: ", err.Error())
	}

	result, erro := json.MarshalIndent(res, "", "    ")
	if erro != nil {
		log.Println("Cannot Marshal JSON because ", erro.Error())
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(result)

}

// err_nsq := hlm.NSQ.Publish("Counter_Visitor_NSQ", res)
// if err_nsq != nil {
// 	log.Println("Failed to publish NSQ message. Error: ", err_nsq)
// }

func (hlm *HelloWorldModule) ShowIndex(w http.ResponseWriter, r *http.Request) {
	hlm.stats.Add(1)

	key := "counter"
	pool := hlm.Redis.Get()

	//get the value of redis
	//error handlingggggg
	value, er := redis.Int64(pool.Do("GET", key))
	if er != nil {
		_, err_set := redis.Int64(pool.Do("SET", key, 1))
		value = 1
		if err_set != nil {
			log.Printf("Failed to Set key %s . Error: %s\n", key, err_set.Error())

		}
	}

	//increment the template counter
	myMap := map[string]int64{
		"counter": value + 1,
	}
	log.Println(myMap)

	renderMap := map[string]int64{
		"counter": value,
	}

	//increment the redist counter
	// value = value + 1
	// pool.Do("SET", key, value)

	nsqq := nsqq.NewNSQModule()
	nsqq.PrintMessage()

	err := hlm.render.ExecuteTemplate(w, "home.html", renderMap)
	if err != nil {
		log.Println("Gagal Render Template because: ", err.Error())
	}
}
