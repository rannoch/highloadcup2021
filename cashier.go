package main

import (
	"fmt"
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
			for {
				cash, _, err := cashier.client.Cash(fmt.Sprintf("\"%s\"", treasure))
				if err == nil {
					AddToWallet(cash)
					break
				}
			}
		}
	}
}
