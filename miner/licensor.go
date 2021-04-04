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

	licenses []model.License

	licensesCond  *sync.Cond
	licensesMutex sync.RWMutex

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

func (licensor *Licensor) Start() {
	for i := 0; i < licensor.workerCount; i++ {
		go licensor.issueLicense()
	}
}

func (licensor *Licensor) issueLicense() {
	for {
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
			licensor.licensesCond.Broadcast()
			licensor.licensesMutex.Unlock()
		}
		if respCode == 409 {
			continue
			//time.Sleep(5 * time.Millisecond)
		}
	}
}
