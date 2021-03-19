package miner

import (
	"encoding/json"
	"github.com/rannoch/highloadcup2021/api_client"
	"github.com/rannoch/highloadcup2021/miner/model"
	"sync"
	"time"
)

type Licensor struct {
	client *api_client.Client

	getLicenseChan   chan model.License
	licenseIssueChan chan interface{}

	workerCount int

	stat     LicensorStat
	showStat bool
}

type LicensorStat struct {
	LicensesUsedMap map[int32]int32
	ResponseCodes   map[int]int

	Mutex sync.RWMutex
}

func NewLicensor(
	client *api_client.Client,
	getLicenseChan chan model.License,
	workerCount int,
	showStat bool,
) *Licensor {
	l := &Licensor{client: client, getLicenseChan: getLicenseChan}
	l.stat.LicensesUsedMap = make(map[int32]int32)
	l.stat.ResponseCodes = make(map[int]int)

	l.licenseIssueChan = make(chan interface{}, 10-workerCount)
	l.workerCount = workerCount
	l.showStat = showStat

	return l
}

func (licensor *Licensor) PrintStat(duration time.Duration) {
	licensor.stat.Mutex.RLock()
	println("Licenses stat after: " + duration.String())
	licensesUsedMapJson, _ := json.Marshal(licensor.stat.LicensesUsedMap)
	println("Licenses used:", string(licensesUsedMapJson))
	licensesResponseCodesJson, _ := json.Marshal(licensor.stat.ResponseCodes)
	println("Licenses response codes:", string(licensesResponseCodesJson))

	println()
	licensor.stat.Mutex.RUnlock()
}

func (licensor *Licensor) queueLicense() {
	licensor.licenseIssueChan <- true
}

func (licensor *Licensor) GetLicense() model.License {
	licensor.queueLicense()

	return <-licensor.getLicenseChan
}

func (licensor *Licensor) Init() {
	for i := 0; i < licensor.workerCount; i++ {
		go licensor.issueLicense()
	}
}

func (licensor *Licensor) Start() {
	for i := 0; i < licensor.workerCount; i++ {
		licensor.queueLicense()
	}
}

func (licensor *Licensor) issueLicense() {
	for {
		select {
		case <-licensor.licenseIssueChan:
			coinsFromWallet := PopCoinsFromWallet()

			for {
				license, respCode, _ := licensor.client.IssueLicense(coinsFromWallet)
				if licensor.showStat {
					licensor.stat.Mutex.Lock()
					licensor.stat.ResponseCodes[respCode]++
					licensor.stat.Mutex.Unlock()
				}

				if respCode == 200 {
					if licensor.showStat {
						licensor.stat.Mutex.Lock()
						licensor.stat.LicensesUsedMap[license.DigAllowed]++
						licensor.stat.Mutex.Unlock()
					}

					licensor.getLicenseChan <- license
					break
				}
				if respCode == 409 {
					continue
				}
			}
		}
	}
}
