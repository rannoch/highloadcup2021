package main

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"time"
)

func main() {
	getenv := os.Getenv("ADDRESS")
	if getenv == "" {
		getenv = "localhost"
	}

	urlParsed, err := url.Parse("http://" + getenv + ":8000")
	if err != nil {
		log.Fatal(err)
	}
	urlParsed.Scheme = "http"

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	client := &fasthttp.Client{
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		MaxConnsPerHost: 100,
	}
	apiClient := NewClient(client, urlParsed.String())

	miner := NewMiner(apiClient)

	err = miner.Start()
	fmt.Println(err)
}
