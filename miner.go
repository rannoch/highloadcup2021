package main

import (
	"fmt"
	openapi "github.com/rannoch/highloadcup2021/client"
	"sync"
	"time"
)

type licenseId = int32

type Miner struct {
	balance  openapi.Balance
	licenses map[licenseId]openapi.License

	explorers []*Explorer
	diggers   []*Digger

	cashierChan chan string

	client *Client
	mutex  sync.RWMutex
}

func NewMiner(client *Client, diggersCount, explorersCount int) *Miner {
	m := &Miner{client: client}
	m.licenses = make(map[licenseId]openapi.License)

	var treasureCoordChan = make(chan TreasureInfo, 100)
	m.cashierChan = make(chan string, 1000)

	for i := 0; i < diggersCount; i++ {
		m.diggers = append(m.diggers, NewDigger(client, m, treasureCoordChan, m.cashierChan))
	}

	const xStep = 1750
	const yStep = 700

	for i := 0; i < explorersCount; i++ {
		area := openapi.Area{
			PosX:  int32(i%2) * xStep,
			PosY:  int32(i/2) * yStep,
			SizeX: xStep,
			SizeY: yStep,
		}

		m.explorers = append(m.explorers, NewExplorer(client, area, treasureCoordChan))
	}

	return m
}

type Coord struct {
	X, Y int32
}

type Explorer struct {
	client *Client

	area             openapi.Area
	treasureInfoChan chan<- TreasureInfo
}

func NewExplorer(client *Client, area openapi.Area, treasureCoordChan chan<- TreasureInfo) *Explorer {
	return &Explorer{client: client, area: area, treasureInfoChan: treasureCoordChan}
}

func (e *Explorer) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	const step = 5

	for x := e.area.PosX; x < e.area.PosX+e.area.SizeX; x += step {
		for y := e.area.PosY; y < e.area.PosY+e.area.SizeY; y += step {
			area, responseCode, err := e.client.ExploreArea(openapi.Area{
				PosX:  x,
				PosY:  y,
				SizeX: step,
				SizeY: step,
			})
			if err != nil || responseCode != 200 || area.Amount == 0 {
				continue
			}

			left := area.Amount

		cellByCellSearch:
			for xSingleCell := x; xSingleCell < x+step; xSingleCell++ {
				for ySingleCell := y; ySingleCell < y+step; ySingleCell++ {
					area, responseCode, err := e.client.ExploreArea(openapi.Area{
						PosX:  xSingleCell,
						PosY:  ySingleCell,
						SizeX: 1,
						SizeY: 1,
					})
					if err != nil || responseCode != 200 || area.Amount == 0 {
						continue
					}

					e.treasureInfoChan <- TreasureInfo{
						Coord: Coord{
							X: xSingleCell,
							Y: ySingleCell,
						},
						Amount: area.Amount,
					}

					left = left - area.Amount

					if left == 0 {
						break cellByCellSearch
					}
				}
			}
		}
	}
}

type TreasureInfo struct {
	Coord  Coord
	Amount int32
}

type Digger struct {
	client *Client
	miner  *Miner

	treasureInfoChan <-chan TreasureInfo
	cashierChan      chan<- string
}

func NewDigger(client *Client, miner *Miner, treasureInfoChan <-chan TreasureInfo, cashierChan chan<- string) *Digger {
	return &Digger{client: client, miner: miner, treasureInfoChan: treasureInfoChan, cashierChan: cashierChan}
}

func (digger *Digger) Start() {
	for treasureInfo := range digger.treasureInfoChan {
		var depth int32 = 1
		left := treasureInfo.Amount

		for depth <= 10 && left > 0 {
			// get license
			licenseId := digger.miner.getRandomLicense()

			// dig
			treasures, digRespCode, err := digger.client.Dig(openapi.Dig{
				LicenseID: licenseId,
				PosX:      treasureInfo.Coord.X,
				PosY:      treasureInfo.Coord.Y,
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
					left--
				}
			}
		}
	}
}

func (miner *Miner) licenseIssuer() {
	for {
		miner.mutex.RLock()
		licensesLen := len(miner.licenses)
		miner.mutex.RUnlock()

		if licensesLen < 10 {
			license, responseCode, err := miner.client.IssueLicense(miner.popCoin())
			if responseCode >= 500 {
				continue
			}

			if responseCode == 409 {
				miner.syncLicenseList()
				continue
			}

			if err != nil {
				continue
			}

			miner.mutex.Lock()
			miner.licenses[license.Id] = license
			miner.mutex.Unlock()
		}
	}
}

func (miner *Miner) cashier(c <-chan string) {
	for {
		select {
		case treasure := <-c:
			for {
				cash, _, err := miner.client.Cash(fmt.Sprintf("\"%s\"", treasure))
				if err != nil {
					continue
				}

				miner.mutex.Lock()
				miner.balance.Wallet = append(miner.balance.Wallet, cash...)

				if len(miner.balance.Wallet) > 100 {
					miner.balance.Wallet = miner.balance.Wallet[:100]
				}
				miner.balance.Balance += int32(len(cash))
				miner.mutex.Unlock()
				break
			}
		}
	}
}

func (miner *Miner) syncLicenseList() {
	for {
		licenses, _, err := miner.client.ListLicenses()
		if err == nil {
			miner.mutex.Lock()
			miner.licenses = make(map[licenseId]openapi.License, len(licenses))

			for _, license := range licenses {
				miner.licenses[license.Id] = license
			}
			miner.mutex.Unlock()
			break
		}
	}
}

func (miner *Miner) getRandomLicense() licenseId {
	for {
		miner.mutex.RLock()
		licensesLen := len(miner.licenses)
		miner.mutex.RUnlock()

		if licensesLen == 0 {
			continue
		}

		var max int32 = 0
		var maxLicenseId int32 = 0

		for licenseId, license := range miner.getLicenses() {
			if license.DigAllowed-license.DigUsed > max {
				max = license.DigAllowed - license.DigUsed
				maxLicenseId = licenseId
			}
		}

		return maxLicenseId
	}
}

func (miner *Miner) getLicenses() map[licenseId]openapi.License {
	miner.mutex.RLock()
	defer miner.mutex.RUnlock()

	var licenses = make(map[licenseId]openapi.License, len(miner.licenses))

	for id, license := range miner.licenses {
		licenses[id] = license
	}

	return licenses
}

func (miner *Miner) useLicense(id licenseId) {
	miner.mutex.Lock()
	defer miner.mutex.Unlock()
	license := miner.licenses[id]
	license.DigUsed++

	if license.DigUsed >= license.DigAllowed {
		delete(miner.licenses, id)
	} else {
		miner.licenses[id] = license
	}
}

func (miner *Miner) deleteLicense(id licenseId) {
	miner.mutex.Lock()
	defer miner.mutex.Unlock()
	delete(miner.licenses, id)
}

func (miner *Miner) popCoin() []int32 {
	miner.mutex.RLock()
	wallenLen := len(miner.balance.Wallet)
	miner.mutex.RUnlock()

	if wallenLen == 0 {
		return []int32{}
	}

	miner.mutex.RLock()
	coin := miner.balance.Wallet[wallenLen-1]
	miner.mutex.RUnlock()

	miner.mutex.Lock()
	miner.balance.Wallet = miner.balance.Wallet[:wallenLen-1]
	miner.mutex.Unlock()

	return []int32{coin}
}

func (miner *Miner) healthCheck() {
	for {
		responseCode, _ := miner.client.HealthCheck()
		if responseCode == 200 {
			break
		}

		time.Sleep(1 * time.Second)
	}
}

func (miner *Miner) Start() error {
	miner.healthCheck()

	go miner.licenseIssuer()

	go miner.cashier(miner.cashierChan)

	wg := sync.WaitGroup{}

	wg.Add(len(miner.explorers))

	for _, explorer := range miner.explorers {
		go explorer.Start(&wg)
	}

	wg.Add(len(miner.diggers))
	for _, digger := range miner.diggers {
		go digger.Start()
	}

	wg.Wait()

	return nil
}
