package main

import (
	"flag"
	"log"
	"net/http"

	"hello/hello"

	grace "gopkg.in/tokopedia/grace.v1"
	logging "gopkg.in/tokopedia/logging.v1"
)

func main() {

	flag.Parse()
	logging.LogInit()

	debug := logging.Debug.Println

	debug("app started") // message will not appear unless run with -debug switch

	hwm := hello.NewNSQModule()

	//http.HandleFunc("/hello", hwm.SayHelloWorld)

	//FOR TRAINING
	http.HandleFunc("/index", hwm.ShowIndex)
	http.HandleFunc("/retrieve", hwm.GetMultiDataFromDatabase)

	go logging.StatsLog()

	log.Fatal(grace.Serve(":9100", nil))
}
