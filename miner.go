package main

import (
	"fmt"
	"github.com/rannoch/highloadcup2021/model"
	"sync"
	"time"
)

var wallet []int32
var walletMutex sync.RWMutex

func AddToWallet(v []int32) {
	walletMutex.Lock()
	wallet = append(wallet, v...)
	if len(wallet) > 100 {
		wallet = wallet[:100]
	}

	walletMutex.Unlock()
}

func PopCoinFromWallet() []int32 {
	walletMutex.RLock()
	walletLen := len(wallet)
	walletMutex.RUnlock()

	if walletLen == 0 {
		return []int32{}
	}

	walletMutex.Lock()
	defer walletMutex.Unlock()
	coin := []int32{wallet[0]}
	wallet = wallet[1:]
	return coin
}

type Miner struct {
	balance model.Balance

	explorer *Explorer
	licensor *Licensor

	diggers []*Digger

	cashiers []*Cashier

	client *Client
}

func NewMiner(client *Client, diggersCount, cashiersCount, explorersCount, licensorsCount int) *Miner {
	m := &Miner{client: client}

	var treasureCoordChan = make(chan model.Report, 10)
	var cashierChan = make(chan string, 10)
	var licensorChan = make(chan model.License)

	for i := 0; i < diggersCount; i++ {
		m.diggers = append(m.diggers, NewDigger(client, treasureCoordChan, cashierChan, licensorChan))
	}

	m.explorer = NewExplorer(client, treasureCoordChan, explorersCount)

	for i := 0; i < cashiersCount; i++ {
		m.cashiers = append(m.cashiers, NewCashier(client, cashierChan))
	}

	m.licensor = NewLicensor(client, licensorChan, licensorsCount)

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

	go miner.licensor.Start()

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
