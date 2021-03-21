package miner

import (
	"encoding/json"
	"github.com/rannoch/highloadcup2021/api_client"
	"github.com/rannoch/highloadcup2021/miner/model"
	"strconv"
	"sync"
	"time"
)

var DiggerStat = diggerStat{
	ResponseCodes: make(map[int]int),
}

type diggerStat struct {
	mutex                    sync.RWMutex
	TreasuresTotal           int
	CashierChanWaitTimeTotal time.Duration
	GetLicenseStartTimeTotal time.Duration
	ResponseCodes            map[int]int
}

func (d *diggerStat) printStat(duration time.Duration) {
	d.mutex.RLock()
	println("Digger treasures total after " + duration.String() + " " + strconv.Itoa(d.TreasuresTotal))
	println("Digger wait for cashier total after " + duration.String() + " " + d.CashierChanWaitTimeTotal.String())
	println("Digger wait for license total after " + duration.String() + " " + d.GetLicenseStartTimeTotal.String())
	responseCodesJson, _ := json.Marshal(d.ResponseCodes)
	println("Digger response codes: " + string(responseCodesJson))
	d.mutex.RUnlock()
	println()
}

type Digger struct {
	client *api_client.Client

	treasureReportChan       <-chan model.Report
	treasureReportChanUrgent <-chan model.Report

	cashierChan       chan<- model.Treasure
	cashierChanUrgent chan<- model.Treasure

	licensor *Licensor

	license model.License

	showStat bool
}

func NewDigger(
	client *api_client.Client,
	treasureReportChan <-chan model.Report,
	treasureReportChanUrgent <-chan model.Report,
	cashierChan chan<- model.Treasure,
	cashierChanUrgent chan<- model.Treasure,
	licensor *Licensor,
	showStat bool,
) *Digger {
	return &Digger{
		client:                   client,
		treasureReportChan:       treasureReportChan,
		treasureReportChanUrgent: treasureReportChanUrgent,
		cashierChan:              cashierChan,
		cashierChanUrgent:        cashierChanUrgent,
		licensor:                 licensor,
		showStat:                 showStat,
	}
}

func (digger Digger) hasActiveLicense() bool {
	return digger.license.DigAllowed > digger.license.DigUsed
}

func (digger *Digger) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	var report model.Report

	for {
		select {
		case report = <-digger.treasureReportChanUrgent:
			digger.dig(report)
		default:
			select {
			case report = <-digger.treasureReportChanUrgent:
				digger.dig(report)
			case report = <-digger.treasureReportChan:
				digger.dig(report)
			}
		}
	}
}

func (digger *Digger) dig(report model.Report) {
	var sendingToCashierStartTime time.Time
	var getLicenseStartTime time.Time

	var left = report.Amount

	var depth int32 = 1

	for depth <= 10 && left > 0 {
		// get license
		if digger.showStat {
			getLicenseStartTime = time.Now()
		}
		for !digger.hasActiveLicense() {
			digger.license = digger.licensor.GetLicense()
		}
		if digger.showStat {
			DiggerStat.mutex.Lock()
			DiggerStat.GetLicenseStartTimeTotal += time.Now().Sub(getLicenseStartTime)
			DiggerStat.mutex.Unlock()
		}

		// dig
		treasureIds, digRespCode, _ := digger.client.Dig(
			model.Dig{
				LicenseID: digger.license.Id,
				PosX:      report.Area.PosX,
				PosY:      report.Area.PosY,
				Depth:     depth,
			},
		)

		if digger.showStat {
			DiggerStat.mutex.Lock()
			DiggerStat.ResponseCodes[digRespCode]++
			DiggerStat.mutex.Unlock()
		}

		if digRespCode == 200 && len(treasureIds) == 0 {
			continue
		}

		if digRespCode == 403 {
			continue
		}

		if digRespCode >= 500 {
			continue
		}

		depth++
		digger.license.DigUsed++

		if !digger.hasActiveLicense() {
			digger.licensor.LicenseExpired()
		}

		if digRespCode == 404 {
			continue
		}

		if digRespCode == 422 {
			continue
		}

		if len(treasureIds) > 0 {
			if digger.showStat {
				DiggerStat.mutex.Lock()
				DiggerStat.TreasuresTotal++
				DiggerStat.mutex.Unlock()
			}

			left = left - int32(len(treasureIds))

			if depth < 4 {
				continue
			}

			for i := range treasureIds {
				if digger.showStat {
					sendingToCashierStartTime = time.Now()
				}

				treasure := model.Treasure{
					Id:    treasureIds[i],
					Depth: depth - 1,
				}

				cashierChan := digger.cashierChan

				if treasure.Depth > 5 {
					cashierChan = digger.cashierChanUrgent
				}

				select {
				case cashierChan <- treasure:
					if digger.showStat {
						DiggerStat.mutex.Lock()
						DiggerStat.CashierChanWaitTimeTotal += time.Now().Sub(sendingToCashierStartTime)
						DiggerStat.mutex.Unlock()
					}
				}
			}
		}
	}
}

func (digger *Digger) getLicense() model.License {
	coinsFromWallet := PopCoinsFromWallet()

	for {
		license, respCode, _ := digger.client.IssueLicense(coinsFromWallet)
		if respCode == 409 {
			continue
		}

		if respCode == 200 {
			return license
		}
	}
}
