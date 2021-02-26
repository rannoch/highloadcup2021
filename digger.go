package main

import (
	openapi "github.com/rannoch/highloadcup2021/client"
)

type Digger struct {
	client *Client
	miner  *Miner

	treasureReportChan <-chan openapi.Report
	cashierChan        chan<- string
}

func NewDigger(client *Client, miner *Miner, treasureInfoChan <-chan openapi.Report, cashierChan chan<- string) *Digger {
	return &Digger{client: client, miner: miner, treasureReportChan: treasureInfoChan, cashierChan: cashierChan}
}

func (digger *Digger) Start() {
	for report := range digger.treasureReportChan {
		var left = report.Amount

		for x := report.Area.PosX; x < report.Area.PosX+report.Area.SizeX; x++ {
			for y := report.Area.PosY; y < report.Area.PosY+report.Area.SizeY; y++ {
				var depth int32 = 1

				for depth <= 10 && left > 0 {
					// get license
					licenseId := digger.miner.getRandomLicense()

					// dig
					treasures, digRespCode, err := digger.client.Dig(openapi.Dig{
						LicenseID: licenseId,
						PosX:      x,
						PosY:      y,
						Depth:     depth,
					})

					if digRespCode == 403 {
						digger.miner.deleteLicense(licenseId)
						continue
					}

					if digRespCode >= 500 {
						continue
					}

					depth++
					digger.miner.useLicense(licenseId)

					if digRespCode == 404 {
						continue
					}

					if digRespCode == 422 {
						continue
					}

					if err != nil {
						continue
					}

					if len(treasures) > 0 {
						for _, treasure := range treasures {
							digger.cashierChan <- treasure
						}

						left = left - int32(len(treasures))
						break
					}
				}
			}
		}
	}
}
