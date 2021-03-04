package main

import (
	"github.com/rannoch/highloadcup2021/model"
	"strconv"
	"sync"
	"time"
)

var DiggerStat = diggerStat{}

type diggerStat struct {
	mutex                    sync.RWMutex
	TreasuresTotal           int
	CashierChanWaitTimeTotal time.Duration
}

func (d *diggerStat) printDiggerStat(duration time.Duration) {
	d.mutex.RLock()
	println("Digger treasures total after " + duration.String() + " " + strconv.Itoa(d.TreasuresTotal))
	println("Digger wait for cashier total after " + duration.String() + " " + d.CashierChanWaitTimeTotal.String())
	d.mutex.RUnlock()
	println()
}

type Digger struct {
	client *Client

	treasureReportChan       <-chan model.Report
	treasureReportChanUrgent <-chan model.Report

	cashierChan    chan<- string
	getLicenseChan <-chan model.License

	license model.License

	showStat bool
}

func NewDigger(
	client *Client,
	treasureReportChan <-chan model.Report,
	treasureReportChanUrgent <-chan model.Report,
	cashierChan chan<- string,
	getLicenseChan <-chan model.License,
	showStat bool,
) *Digger {
	return &Digger{
		client:                   client,
		treasureReportChan:       treasureReportChan,
		treasureReportChanUrgent: treasureReportChanUrgent,
		cashierChan:              cashierChan,
		getLicenseChan:           getLicenseChan,
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
	var left = report.Amount

	var depth int32 = 1

	for depth <= 10 && left > 0 {
		// get license
		if !digger.hasActiveLicense() {
			digger.license = <-digger.getLicenseChan
		}

		// dig
		treasures, digRespCode, _ := digger.client.Dig(model.Dig{
			LicenseID: digger.license.Id,
			PosX:      report.Area.PosX,
			PosY:      report.Area.PosY,
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
			if digger.showStat {
				DiggerStat.mutex.Lock()
				DiggerStat.TreasuresTotal++
				DiggerStat.mutex.Unlock()
			}

			for i := range treasures {
				if digger.showStat {
					sendingToCashierStartTime = time.Now()
				}
				select {
				case digger.cashierChan <- treasures[i]:
					if digger.showStat {
						DiggerStat.mutex.Lock()
						DiggerStat.CashierChanWaitTimeTotal += time.Now().Sub(sendingToCashierStartTime)
						DiggerStat.mutex.Unlock()
					}
				}
			}

			left = left - int32(len(treasures))
		}
	}
}
