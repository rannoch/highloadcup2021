package main

import (
	"context"
	"fmt"
	"github.com/antihax/optional"
	openapi "github.com/rannoch/highloadcup2021/client"
	"sync"
	"time"
)

type licenseId = int32

type Miner struct {
	balance  openapi.Balance
	licenses map[licenseId]openapi.License

	client *openapi.APIClient
	mutex  sync.Mutex
}

func NewMiner(client *openapi.APIClient) *Miner {
	m := &Miner{client: client}
	m.licenses = make(map[licenseId]openapi.License)

	return m
}

func (miner *Miner) licenseIssuer(ctx context.Context) {
	for {
		if len(miner.licenses) < 10 {
			license, response, err := miner.client.DefaultApi.IssueLicense(ctx, &openapi.IssueLicenseOpts{Args: optional.NewInterface(miner.popCoin())})
			if response != nil && response.StatusCode >= 500 {
				continue
			}

			if response != nil && response.StatusCode == 409 {
				miner.syncLicenseList(ctx)
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

func (miner *Miner) syncLicenseList(ctx context.Context) {
	for {
		licenses, _, err := miner.client.DefaultApi.ListLicenses(ctx)
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
		if len(miner.licenses) == 0 {
			continue
		}

		miner.mutex.Lock()
		var max int32 = 0
		var maxLicenseId int32 = 0

		for licenseId, license := range miner.licenses {
			if license.DigAllowed-license.DigUsed > max {
				max = license.DigAllowed - license.DigUsed
				maxLicenseId = licenseId
			}
		}

		miner.mutex.Unlock()

		return maxLicenseId
	}
}

func (miner *Miner) useLicense(id licenseId) {
	miner.mutex.Lock()
	license := miner.licenses[id]
	license.DigUsed++

	if license.DigUsed >= license.DigAllowed {
		delete(miner.licenses, id)
	} else {
		miner.licenses[id] = license
	}
	miner.mutex.Unlock()
}

func (miner *Miner) deleteLicense(id licenseId) {
	miner.mutex.Lock()
	delete(miner.licenses, id)
	miner.mutex.Unlock()
}

func (miner *Miner) popCoin() []int32 {
	if len(miner.balance.Wallet) == 0 {
		return []int32{}
	}

	coin := miner.balance.Wallet[len(miner.balance.Wallet)-1]

	miner.mutex.Lock()
	miner.balance.Wallet = miner.balance.Wallet[:len(miner.balance.Wallet)-1]
	miner.mutex.Unlock()

	return []int32{coin}
}

func (miner *Miner) healthCheck(ctx context.Context) {
	for {
		_, response, _ := miner.client.DefaultApi.HealthCheck(ctx)
		if response != nil && response.StatusCode == 200 {
			break
		}

		time.Sleep(1 * time.Second)
	}
}

func (miner *Miner) startWorker(
	ctx context.Context,
	fromX, toX, fromY, toY, size int32,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for x := fromX; x < toX; x += size {
		for y := fromY; y < toY; y += size {
			area, response, err := miner.client.DefaultApi.ExploreArea(ctx, openapi.Area{
				PosX:  x,
				PosY:  y,
				SizeX: size,
				SizeY: size,
			})
			if err != nil || response.StatusCode != 200 || area.Amount == 0 {
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
							treasures, digResp, err := miner.client.DefaultApi.Dig(ctx, openapi.Dig{
								LicenseID: licenseId,
								PosX:      xDig,
								PosY:      yDig,
								Depth:     depth,
							})

							if digResp != nil && digResp.StatusCode == 403 {
								miner.deleteLicense(licenseId)
								continue
							}

							if digResp != nil && digResp.StatusCode >= 500 {
								continue
							}

							depth++
							miner.useLicense(licenseId)

							if digResp != nil && digResp.StatusCode == 404 {
								continue
							}

							if digResp != nil && digResp.StatusCode == 422 {
								continue
							}

							if err != nil {
								continue
							}

							if len(treasures) > 0 {
								for _, treasure := range treasures {
									for {
										cash, _, err := miner.client.DefaultApi.Cash(ctx, fmt.Sprintf("\"%s\"", treasure))
										if err != nil {
											continue
										}
										left = left - 1

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
					}
				}
			}
		}
	}

}

func (miner *Miner) Start(ctx context.Context) error {
	miner.healthCheck(ctx)

	go miner.licenseIssuer(ctx)

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

		go miner.startWorker(ctx, fromX, toX, fromY, toY, 1, &wg)
	}

	wg.Wait()

	return nil
}
