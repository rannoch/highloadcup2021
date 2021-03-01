package main

import (
	"github.com/rannoch/highloadcup2021/model"
	"sync"
)

type Digger struct {
	client *Client

	treasureReportChan <-chan model.Report
	cashierChan        chan<- string
	getLicenseChan     <-chan model.License

	license model.License
}

func NewDigger(
	client *Client,
	treasureReportChan <-chan model.Report,
	cashierChan chan<- string,
	getLicenseChan <-chan model.License,
) *Digger {
	return &Digger{
		client:             client,
		treasureReportChan: treasureReportChan,
		cashierChan:        cashierChan,
		getLicenseChan:     getLicenseChan,
	}
}

func (digger *Digger) hasActiveLicense() bool {
	return digger.license.DigAllowed > digger.license.DigUsed
}

func (digger *Digger) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	for report := range digger.treasureReportChan {
		var left = report.Amount

		for x := report.Area.PosX; x < report.Area.PosX+report.Area.SizeX; x++ {
			for y := report.Area.PosY; y < report.Area.PosY+report.Area.SizeY; y++ {
				var depth int32 = 1

				for depth <= 10 && left > 0 {
					// get license
					if !digger.hasActiveLicense() {
						digger.license = <-digger.getLicenseChan
					}

					// dig
					treasures, digRespCode, _ := digger.client.Dig(model.Dig{
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
