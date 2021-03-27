package main

import (
	"encoding/json"
	"fmt"
	"github.com/rannoch/highloadcup2021/api_client"
	"github.com/rannoch/highloadcup2021/miner/model"
	"github.com/valyala/fasthttp"
	"log"
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

	fastHttpClient := &fasthttp.Client{}
	apiClient := api_client.NewClient(fastHttpClient, urlParsed.String())

	//treasuresDepthMap := make(map[string][]model.Treasure)

	var treasures []model.Treasure

	err = json.Unmarshal(TreasuresDepthMap, &treasures)
	if err != nil {
		panic(err)
	}

	//
	//sort.Slice(treasures, func(i, j int) bool {
	//	return treasures[i].Depth < treasures[j].Depth
	//})

	license := model.License{}

	healthCheck(apiClient)

	for _, treasure := range treasures {
		// dig
		var depth int32 = 1
		var left = 1

		for depth <= 10 && left > 0 {
			if license.DigUsed >= license.DigAllowed {
				license = getLicense(apiClient)
			}

			treasureIds, digRespCode, _ := apiClient.Dig(model.Dig{
				LicenseID: license.Id,
				PosX:      treasure.PosX,
				PosY:      treasure.PosY,
				Depth:     depth,
			})

			if (digRespCode == 200 && len(treasureIds) == 0) || digRespCode == 403 || digRespCode >= 500 {
				continue
			}

			depth++
			license.DigUsed++

			if digRespCode == 404 || digRespCode == 422 {
				continue
			}

			if len(treasureIds) > 0 {
				for _, treasureId := range treasureIds {
					for {
						coins, _, err := apiClient.Cash(`"` + treasureId + `"`)
						if err == nil {
							println(treasure.Depth, len(coins))

							break
						}
					}
				}

				left--
			}
		}
	}
}

func getLicense(client *api_client.Client) model.License {
	for {
		license, respCode, _ := client.IssueLicense([]int32{})
		if respCode == 200 {
			return license
		}
	}
}

func healthCheck(client *api_client.Client) {
	fmt.Println("healthCheck started")

	for {
		responseCode, _ := client.HealthCheck()
		if responseCode == 200 {
			break
		}

		time.Sleep(1 * time.Millisecond)
	}

	fmt.Println("healthCheck passed")
}
