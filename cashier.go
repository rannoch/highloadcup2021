package main

import (
	"sync"
)

type Cashier struct {
	client *Client

	treasureChan <-chan string
}

func NewCashier(client *Client, treasureChan <-chan string) *Cashier {
	return &Cashier{client: client, treasureChan: treasureChan}
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
		cash, _, err := cashier.client.Cash(`"` + treasure + `"`)
		if err == nil {
			AddToWallet(cash)
			return
		}
	}
}
