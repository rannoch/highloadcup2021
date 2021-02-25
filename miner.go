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

	client *Client
	mutex  sync.RWMutex
}

func NewMiner(client *Client) *Miner {
	m := &Miner{client: client}
	m.licenses = make(map[licenseId]openapi.License)

	return m
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

func (miner *Miner) startWorker(
	fromX, toX, fromY, toY, size int32,
	wg *sync.WaitGroup,
	cashierChan chan<- string,
) {
	defer wg.Done()

	for x := fromX; x < toX; x += size {
		for y := fromY; y < toY; y += size {
			area, responseCode, err := miner.client.ExploreArea(openapi.Area{
				PosX:  x,
				PosY:  y,
				SizeX: size,
				SizeY: size,
			})
			if err != nil || responseCode != 200 || area.Amount == 0 {
				continue
			}

			left := area.Amount

		main:
			for left > 0 {
				for xDig := x; xDig < x+size; xDig++ {
					for yDig := y; yDig < y+size; yDig++ {
						var depth int32 = 1
						if left == 0 {
							break main
						}

						for depth <= 10 {
							// get license
							licenseId := miner.getRandomLicense()

							// dig
							treasures, digRespCode, err := miner.client.Dig(openapi.Dig{
								LicenseID: licenseId,
								PosX:      xDig,
								PosY:      yDig,
								Depth:     depth,
							})

							if digRespCode == 403 {
								miner.deleteLicense(licenseId)
								continue
							}

							if digRespCode >= 500 {
								continue
							}

							depth++
							miner.useLicense(licenseId)

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
									cashierChan <- treasure
									left--
								}
							}
						}
					}
				}
			}
		}
	}

}

func (miner *Miner) Start() error {
	miner.healthCheck()

	go miner.licenseIssuer()

	var cashierChan = make(chan string, 1000)

	go miner.cashier(cashierChan)

	wg := sync.WaitGroup{}

	const xStep = 1750
	const yStep = 700

	var fromX, toX, fromY, toY int32

	for i := 0; i < 10; i++ {
		wg.Add(1)

		fromX = int32(i%2) * xStep
		toX = int32(i%2+1) * xStep

		fromY = int32(i/2) * yStep
		toY = int32(i/2+1) * yStep

		go miner.startWorker(fromX, toX, fromY, toY, 1, &wg, cashierChan)
	}

	wg.Wait()

	return nil
}
