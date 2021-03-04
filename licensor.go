package main

import (
	"fmt"
	"github.com/rannoch/highloadcup2021/model"
	"sync"
	"time"
)

type Licensor struct {
	client *Client

	getLicenseChan   chan<- model.License
	licenseIssueChan chan interface{}

	licenses       []model.License
	licensesIssued int

	workerCount int

	stat     LicensorStat
	showStat bool
}

type LicensorStat struct {
	FreeLicensesIssued int
	PaidLicensesIssued int
	Mutex              sync.RWMutex
}

func (l *LicensorStat) String() string {
	return fmt.Sprintf("Free licenses issued: %d\nPaid licenses issued: %d\n", l.FreeLicensesIssued, l.PaidLicensesIssued)
}

func NewLicensor(client *Client, getLicenseChan chan<- model.License, workerCount int, showStat bool) *Licensor {
	l := &Licensor{client: client, getLicenseChan: getLicenseChan}
	l.licenseIssueChan = make(chan interface{}, 5)
	l.workerCount = workerCount
	l.showStat = showStat

	return l
}

func (licensor *Licensor) PrintStat(duration time.Duration) {
	licensor.stat.Mutex.RLock()
	println("Licenses stat after: " + duration.String())
	println("Paid licenses count:", licensor.stat.PaidLicensesIssued, ", Free licenses count:", licensor.stat.FreeLicensesIssued)
	println()
	licensor.stat.Mutex.RUnlock()
}

func (licensor *Licensor) GetLicense() model.License {
	if len(licensor.licenses) > 0 {
		return licensor.licenses[0]
	}
	return model.License{}
}

func (licensor *Licensor) queueLicense() {
	licensor.licensesIssued++
	licensor.licenseIssueChan <- true
}

func (licensor *Licensor) Start() {
	var addLicenseChan = make(chan model.License)

	for i := 0; i < licensor.workerCount; i++ {
		go func(licenseIssueChan chan interface{}, addLicenseChan chan model.License) {
			for {
				select {
				case <-licenseIssueChan:
					for {
						license, respCode, _ := licensor.client.IssueLicense(PopCoinFromWallet())
						if respCode == 200 {
							addLicenseChan <- license
							break
						}
						if respCode == 409 {
							continue
						}
					}
				}
			}
		}(licensor.licenseIssueChan, addLicenseChan)
	}

	for i := 0; i < licensor.workerCount; i++ {
		licensor.queueLicense()
	}

	for {
		select {
		case licensor.getLicenseChan <- licensor.GetLicense():
			if len(licensor.licenses) > 0 {
				licensor.licenses = licensor.licenses[1:]

				if len(licensor.licenses)+licensor.licensesIssued < (10 - licensor.workerCount) {
					licensor.queueLicense()
				}
			}
		case license := <-addLicenseChan:
			licensor.stat.Mutex.Lock()
			if license.DigAllowed == 5 {
				licensor.stat.PaidLicensesIssued++
			} else {
				licensor.stat.FreeLicensesIssued++
			}
			licensor.stat.Mutex.Unlock()

			licensor.licensesIssued--
			licensor.licenses = append(licensor.licenses, license)
		}
	}
}
