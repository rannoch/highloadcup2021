package main

import (
	"fmt"
	openapi "github.com/rannoch/highloadcup2021/client"
)

type Digger struct {
	client *Client

	wallet  []int32
	license openapi.License

	treasureReportChan <-chan openapi.Report
	cashierChan        chan<- string
}

func NewDigger(client *Client, treasureInfoChan <-chan openapi.Report, cashierChan chan<- string) *Digger {
	return &Digger{client: client, treasureReportChan: treasureInfoChan, cashierChan: cashierChan}
}

func (digger *Digger) hasActiveLicense() bool {
	return digger.license.DigAllowed > digger.license.DigUsed
}

func (digger *Digger) Start() {
	for report := range digger.treasureReportChan {
		var left = report.Amount

		for x := report.Area.PosX; x < report.Area.PosX+report.Area.SizeX; x++ {
			for y := report.Area.PosY; y < report.Area.PosY+report.Area.SizeY; y++ {
				var depth int32 = 1

				for depth <= 10 && left > 0 {
					// get license
					if !digger.hasActiveLicense() {
						var coin = []int32{}
						if len(digger.wallet) > 0 {
							coin = digger.wallet[:1]
							digger.wallet = digger.wallet[1:]
						}

						for {
							license, respCode, _ := digger.client.IssueLicense(coin)
							if respCode == 200 {
								digger.license = license
								break
							}
							if respCode == 409 {
								continue
							}
						}
					}

					// dig
					treasures, digRespCode, _ := digger.client.Dig(openapi.Dig{
						LicenseID: digger.license.Id,
						PosX:      x,
						PosY:      y,
						Depth:     depth,
					})

					if digRespCode == 200 && len(treasures) == 0 {
						continue
					}

					if digRespCode == 403 {
						digger.license.DigAllowed = 0
						continue
					}

					if digRespCode >= 500 {
						continue
					}

					depth++
					digger.license.DigUsed++

					if digRespCode == 404 {
						continue
					}

					if digRespCode == 422 {
						continue
					}

					if len(treasures) > 0 {
						for _, treasure := range treasures {
							for {
								cash, _, err := digger.client.Cash(fmt.Sprintf("\"%s\"", treasure))
								if err == nil {
									digger.wallet = append(digger.wallet, cash...)
									break
								}
							}
						}

						left = left - int32(len(treasures))
						break
					}
				}
			}
		}
	}
}
