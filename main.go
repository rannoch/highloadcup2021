package main

import (
	"context"
	"fmt"
	"github.com/rannoch/highloadcup2021/client"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	configuration := openapi.NewConfiguration()

	getenv := os.Getenv("ADDRESS")
	if getenv == "" {
		getenv = "localhost"
	}

	urlParsed, err := url.Parse("http://" + getenv + ":8000")
	if err != nil {
		log.Fatal(err)
	}
	urlParsed.Scheme = "http"

	configuration.BasePath = urlParsed.String()
	configuration.HTTPClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	//configuration.Debug = true

	apiClient := openapi.NewAPIClient(configuration)
	miner := NewMiner(apiClient)

	err = miner.Start(context.Background())
	fmt.Println(err)
}
