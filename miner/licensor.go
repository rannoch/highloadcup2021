package miner

import (
	"encoding/json"
	"github.com/rannoch/highloadcup2021/api_client"
	"github.com/rannoch/highloadcup2021/miner/model"
	"sync"
	"time"
)

const workerPerLicense = 3

type Licensor struct {
	client *api_client.Client

	licenses       []model.License
	licensesCount  int
	licensesIssued int

	licensesCond     *sync.Cond
	licensesMutex    sync.RWMutex
	licenseIssueChan chan chan struct{}

	licenseIssuerGlobalCancelChan chan struct{}

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
	workerCount int,
	showStat bool,
) *Licensor {
	l := &Licensor{client: client}
	l.stat.LicensesUsedMap = make(map[int32]int32)
	l.stat.ResponseCodes = make(map[int]int)

	l.licensesCond = sync.NewCond(&l.licensesMutex)

	l.licenseIssueChan = make(chan chan struct{}, workerCount)
	l.licenseIssuerGlobalCancelChan = make(chan struct{}, workerCount)

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
	licensor.licensesIssued++
	cancelChan := make(chan struct{}, workerPerLicense)
	for i := 0; i < workerPerLicense; i++ {
		select {
		case licensor.licenseIssueChan <- cancelChan:
		}
	}
}

func (licensor *Licensor) GetLicense() model.License {
	licensor.licensesCond.L.Lock()
	defer licensor.licensesCond.L.Unlock()
	for len(licensor.licenses) == 0 {
		licensor.licensesCond.Wait()
	}

	license := licensor.licenses[0]

	licensor.licenses = licensor.licenses[1:]

	return license
}

func (licensor *Licensor) LicenseExpired() {
	licensor.licensesMutex.Lock()
	licensor.licensesCount--
	licensor.licensesCond.Broadcast()
	licensor.licensesMutex.Unlock()
}

func (licensor *Licensor) Init() {
	for i := 0; i < licensor.workerCount; i++ {
		go licensor.issueLicense()
	}
}

func (licensor *Licensor) Start() {
	go func() {
		for {
			licensor.licensesCond.L.Lock()
			for licensor.licensesCount+licensor.licensesIssued == 10 {
				licensor.licensesCond.Wait()
			}

			for i := 0; i < 10-licensor.licensesCount-licensor.licensesIssued; i++ {
				licensor.queueLicense()
			}

			licensor.licensesCond.L.Unlock()
		}
	}()
}

func (licensor *Licensor) issueLicense() {
	for {
		select {
		case groupCancelChan := <-licensor.licenseIssueChan:
		main:
			for {
				select {
				case <-groupCancelChan:
					break main
				case <-licensor.licenseIssuerGlobalCancelChan:
					break main
				default:
					//licensor.licensesMutex.RLock()
					//licensesCount := len(licensor.licenses)
					//licensor.licensesMutex.RUnlock()
					//if licensesCount == 10 {
					//	break main
					//}

					license, respCode, _ := licensor.client.IssueLicense([]int32{})
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

						licensor.licensesMutex.Lock()
						licensor.licenses = append(licensor.licenses, license)
						licensor.licensesCount++
						licensor.licensesIssued--

						if len(licensor.licenses) == 10 {
							licensor.cancelAllIssuers()
						}

						licensor.licensesCond.Broadcast()
						licensor.licensesMutex.Unlock()

						for i := 0; i < workerPerLicense; i++ {
							select {
							case groupCancelChan <- struct{}{}:
							default:
								//println("licensor 200 break default")
							}
						}
					}
					if respCode == 409 {
						licensor.cancelAllIssuers()
					}
				}
			}
		}
	}
}

func (licensor *Licensor) cancelAllIssuers() {
	for i := 0; i < licensor.workerCount; i++ {
		select {
		case licensor.licenseIssuerGlobalCancelChan <- struct{}{}:
		default:

		}
	}
}
