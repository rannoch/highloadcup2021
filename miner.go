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

	showStat bool
}

func NewMiner(client *Client, diggersCount, cashiersCount, explorersCount, licensorsCount int, showStat bool) *Miner {
	m := &Miner{client: client}
	m.showStat = showStat

	var treasureCoordChan = make(chan model.Report, 10)
	var treasureCoordChanUrgent = make(chan model.Report, 10)

	var cashierChan = make(chan string, 10)
	var licensorChan = make(chan model.License)

	for i := 0; i < diggersCount; i++ {
		m.diggers = append(m.diggers, NewDigger(client, treasureCoordChan, treasureCoordChanUrgent, cashierChan, licensorChan, showStat))
	}

	m.explorer = NewExplorer(client, treasureCoordChan, treasureCoordChanUrgent, explorersCount, showStat)

	for i := 0; i < cashiersCount; i++ {
		m.cashiers = append(m.cashiers, NewCashier(client, cashierChan))
	}

	m.licensor = NewLicensor(client, licensorChan, licensorsCount, showStat)

	return m
}

func (miner *Miner) healthCheck() {
	fmt.Println("healthCheck started")

	for {
		responseCode, _ := miner.client.HealthCheck()
		if responseCode == 200 {
			break
		}

		time.Sleep(1 * time.Millisecond)
	}

	fmt.Println("healthCheck passed")
}

func (miner *Miner) Start() error {
	go miner.licensor.Start()

	wg := sync.WaitGroup{}

	wg.Add(1)

	wg.Add(len(miner.diggers))
	for _, digger := range miner.diggers {
		go digger.Start(&wg)
	}

	wg.Add(len(miner.cashiers))
	for _, cashier := range miner.cashiers {
		go cashier.Start(&wg)
	}

	if miner.showStat {
		go miner.printStat()
	}

	miner.explorer.Init()

	miner.healthCheck()
	go miner.explorer.Start(&wg)

	wg.Wait()

	return nil
}

func (miner *Miner) printStat() {
	startTime := time.Now()

	for {
		select {
		case t := <-time.After(10 * time.Second):
			sub := t.Sub(startTime)

			miner.explorer.PrintStat(sub)
			miner.licensor.PrintStat(sub)
			DiggerStat.printDiggerStat(sub)
		}
	}
}
