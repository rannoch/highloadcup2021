package main

import (
	"fmt"
	openapi "github.com/rannoch/highloadcup2021/client"
	"sync"
	"time"
)

var wallet []int32
var m sync.RWMutex

func AddToWallet(v []int32) {
	m.Lock()
	wallet = append(wallet, v...)
	if len(wallet) > 100 {
		wallet = wallet[:100]
	}

	m.Unlock()
}

func PopCoinFromWallet() []int32 {
	m.RLock()
	walletLen := len(wallet)
	m.RUnlock()

	if walletLen == 0 {
		return []int32{}
	}

	m.Lock()
	defer m.Unlock()
	coin := []int32{wallet[0]}
	wallet = wallet[1:]
	return coin
}

type Miner struct {
	balance openapi.Balance

	explorer *Explorer
	diggers  []*Digger

	cashiers []*Cashier

	client *Client
}

func NewMiner(client *Client, diggersCount, cashiersCount int) *Miner {
	m := &Miner{client: client}

	var treasureCoordChan = make(chan openapi.Report, 10)
	var cashierChan = make(chan string, 10)

	for i := 0; i < diggersCount; i++ {
		m.diggers = append(m.diggers, NewDigger(client, treasureCoordChan, cashierChan))
	}

	m.explorer = NewExplorer(client, treasureCoordChan)

	for i := 0; i < cashiersCount; i++ {
		m.cashiers = append(m.cashiers, NewCashier(client, cashierChan))
	}

	return m
}

func (miner *Miner) healthCheck() {
	fmt.Println("healthCheck started")

	for {
		responseCode, _ := miner.client.HealthCheck()
		if responseCode == 200 {
			break
		}

		time.Sleep(1 * time.Second)
	}

	fmt.Println("healthCheck passed")
}

func (miner *Miner) Start() error {
	miner.healthCheck()

	//go miner.cashier(miner.cashierChan)

	wg := sync.WaitGroup{}

	wg.Add(1)

	go miner.explorer.Start(&wg)

	wg.Add(len(miner.diggers))
	for _, digger := range miner.diggers {
		go digger.Start(&wg)
	}

	wg.Add(len(miner.cashiers))
	for _, cashier := range miner.cashiers {
		go cashier.Start(&wg)
	}

	wg.Wait()

	return nil
}
