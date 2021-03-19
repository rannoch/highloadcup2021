package miner

import (
	"encoding/json"
	"github.com/rannoch/highloadcup2021/api_client"
	"github.com/rannoch/highloadcup2021/miner/model"
	"strconv"
	"sync"
	"time"
)

var CashierStat = cashierStat{
	ResponseCodes:  make(map[int]int),
	DepthCoins:     make(map[int32]int32, 10),
	DepthCoinsMin:  make(map[int32]int32, 10),
	DepthCoinsMax:  make(map[int32]int32, 10),
	DepthTreasures: make(map[int32]int32, 10),
}

type cashierStat struct {
	mutex          sync.RWMutex
	TreasuresTotal int
	ResponseCodes  map[int]int

	DepthCoins    map[int32]int32
	DepthCoinsMin map[int32]int32
	DepthCoinsMax map[int32]int32

	DepthTreasures map[int32]int32
}

func (c *cashierStat) printStat(duration time.Duration) {
	c.mutex.RLock()
	println("Cashier treasures total after " + duration.String() + " " + strconv.Itoa(c.TreasuresTotal))

	responseCodesJson, _ := json.Marshal(c.ResponseCodes)
	println("Cashier response codes: " + string(responseCodesJson))

	depthCoinsJson, _ := json.Marshal(c.DepthCoins)
	println("Cashier depth coins: " + string(depthCoinsJson))

	depthCoinsMinJson, _ := json.Marshal(c.DepthCoinsMin)
	println("Cashier depth coins min: " + string(depthCoinsMinJson))

	depthCoinsMaxJson, _ := json.Marshal(c.DepthCoinsMax)
	println("Cashier depth coins max: " + string(depthCoinsMaxJson))

	depthTreasureJson, _ := json.Marshal(c.DepthTreasures)
	println("Cashier depth treasures: " + string(depthTreasureJson))

	depthCoinsAvg := make(map[int32]int32, 10)
	for depth, coinsAmount := range c.DepthCoins {
		depthCoinsAvg[depth] = coinsAmount / c.DepthTreasures[depth]
	}

	depthCoinsAvgJson, _ := json.Marshal(depthCoinsAvg)
	println("Cashier depth avg coins: " + string(depthCoinsAvgJson))

	c.mutex.RUnlock()
	println()
}

type Cashier struct {
	client *api_client.Client

	treasureChan       <-chan model.Treasure
	treasureChanUrgent <-chan model.Treasure

	showStat bool
}

func NewCashier(
	client *api_client.Client,
	treasureChan, treasureChanUrgent <-chan model.Treasure,
	showStat bool,
) *Cashier {
	return &Cashier{
		client:             client,
		treasureChan:       treasureChan,
		treasureChanUrgent: treasureChanUrgent,
		showStat:           showStat,
	}
}

func (cashier *Cashier) Start(wg *sync.WaitGroup) {
	wg.Done()

	for {
		select {
		case treasure := <-cashier.treasureChanUrgent:
			cashier.cash(treasure)
		default:
			select {
			case treasure := <-cashier.treasureChanUrgent:
				cashier.cash(treasure)
			case treasure := <-cashier.treasureChan:
				cashier.cash(treasure)
			}
		}
	}
}

func (cashier *Cashier) cash(treasure model.Treasure) {
	for {
		coins, httpCode, err := cashier.client.Cash(`"` + treasure.Id + `"`)
		if cashier.showStat {
			CashierStat.mutex.Lock()
			CashierStat.ResponseCodes[httpCode]++
			CashierStat.mutex.Unlock()
		}

		if err == nil {
			treasure.CoinsAmount = int32(len(coins))

			if cashier.showStat {
				CashierStat.mutex.Lock()
				CashierStat.DepthCoins[treasure.Depth] += treasure.CoinsAmount
				if treasure.CoinsAmount < CashierStat.DepthCoinsMin[treasure.Depth] || CashierStat.DepthCoinsMin[treasure.Depth] == 0 {
					CashierStat.DepthCoinsMin[treasure.Depth] = treasure.CoinsAmount
				}

				if treasure.CoinsAmount > CashierStat.DepthCoinsMax[treasure.Depth] {
					CashierStat.DepthCoinsMax[treasure.Depth] = treasure.CoinsAmount
				}

				CashierStat.DepthTreasures[treasure.Depth]++

				CashierStat.TreasuresTotal++
				CashierStat.mutex.Unlock()
			}

			AddToWallet(coins)
			return
		}
	}
}
