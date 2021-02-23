package main

import (
	"context"
	"encoding/json"
	"fmt"
	openapi "github.com/rannoch/highloadcup2021/client"
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
	configuration.Debug = true

	apiClient := openapi.NewAPIClient(configuration)
	areaExaminer := NewAreaExaminer(apiClient)

	areaExaminer.Start(context.Background())
}

const xStep = 100
const yStep = 100
const xMax, yMax = 3500, 3500

type AreaExaminer struct {
	client *openapi.APIClient
}

func NewAreaExaminer(client *openapi.APIClient) *AreaExaminer {
	return &AreaExaminer{client: client}
}

func (areaExaminer AreaExaminer) Start(ctx context.Context) {
	for {
		_, response, _ := areaExaminer.client.DefaultApi.HealthCheck(ctx)
		if response != nil && response.StatusCode == 200 {
			break
		}

		time.Sleep(1 * time.Second)
	}

	var x, y int32
	var result []int32

	for x = 0; x < xMax; x += xStep {
		for y = 0; y < yMax; y += yStep {
			for {
				area, _, err := areaExaminer.client.DefaultApi.ExploreArea(ctx, openapi.Area{
					PosX:  x,
					PosY:  y,
					SizeX: xStep,
					SizeY: yStep,
				})

				if err == nil {
					result = append(result, area.Amount)
					break
				}
			}
		}
	}

	resultJson, _ := json.Marshal(result)

	fmt.Println(string(resultJson))
}
