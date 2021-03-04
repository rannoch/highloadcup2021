package main

import (
	"strconv"
	"sync"
	"time"
)

var CashierStat = cashierStat{
	ResponseCodes: make(map[int]int),
}

type cashierStat struct {
	mutex          sync.RWMutex
	TreasuresTotal int
	ResponseCodes  map[int]int
}

func (c *cashierStat) printStat(duration time.Duration) {
	c.mutex.RLock()
	println("Cashier treasures total after " + duration.String() + " " + strconv.Itoa(c.TreasuresTotal))

	responseCodesJson, _ := json.Marshal(c.ResponseCodes)
	println("Cashier response codes: " + string(responseCodesJson))

	c.mutex.RUnlock()
	println()
}

type Cashier struct {
	client *Client

	treasureChan <-chan string

	showStat bool
}

func NewCashier(client *Client, treasureChan <-chan string, showStat bool) *Cashier {
	return &Cashier{client: client, treasureChan: treasureChan, showStat: showStat}
}

func (cashier *Cashier) Start(wg *sync.WaitGroup) {
	wg.Done()

	for {
		select {
		case treasure := <-cashier.treasureChan:
			cashier.cash(treasure)
		}
	}
}

func (cashier *Cashier) cash(treasure string) {
	for {
		cash, httpCode, err := cashier.client.Cash(`"` + treasure + `"`)
		if cashier.showStat {
			CashierStat.mutex.Lock()
			CashierStat.ResponseCodes[httpCode]++
			CashierStat.mutex.Unlock()
		}

		if err == nil {
			if cashier.showStat {
				CashierStat.mutex.Lock()
				CashierStat.TreasuresTotal++
				CashierStat.mutex.Unlock()
			}

			AddToWallet(cash)
			return
		}
	}
}
