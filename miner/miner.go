package miner

import (
	"fmt"
	"github.com/rannoch/highloadcup2021/api_client"
	"github.com/rannoch/highloadcup2021/miner/model"
	"sync"
	"time"
)

var wallet []int32
var walletMutex sync.RWMutex

func AddToWallet(v []int32) {
	walletMutex.Lock()
	wallet = append(wallet, v...)
	if len(wallet) > 1000 {
		wallet = wallet[:1000]
	}

	walletMutex.Unlock()
}

func PopCoinsFromWallet() []int32 {
	walletMutex.Lock()
	defer walletMutex.Unlock()
	var coins []int32

	switch {
	//case len(wallet) >= 21:
	//	coins = make([]int32, 21)
	//	copy(coins, wallet[:21])
	//	wallet = wallet[21:]
	//case len(wallet) >= 11:
	//	coins = make([]int32, 11)
	//	copy(coins, wallet[:11])
	//	wallet = wallet[11:]
	//case len(wallet) >= 6:
	//	coins = make([]int32, 6)
	//	copy(coins, wallet[:6])
	//	wallet = wallet[6:]
	case len(wallet) >= 1:
		coins = []int32{wallet[0]}
		wallet = wallet[1:]
	default:
		coins = []int32{}
	}
	return coins
}

type Miner struct {
	explorer *Explorer
	licensor *Licensor

	diggers []*Digger

	cashiers []*Cashier

	client *api_client.Client

	showStat bool
}

func NewMiner(
	client *api_client.Client,
	diggersCount, cashiersCount, explorersCount, licensorsCount int,
	showStat bool,
) *Miner {
	m := &Miner{client: client}
	m.showStat = showStat

	var treasureCoordChan = make(chan model.Report, 10)
	var treasureCoordChanUrgent = make(chan model.Report, 10)

	var cashierChan = make(chan model.Treasure, 10000)
	var cashierChanUrgent = make(chan model.Treasure, 10)

	m.licensor = NewLicensor(client, licensorsCount, showStat)

	for i := 0; i < diggersCount; i++ {
		m.diggers = append(
			m.diggers,
			NewDigger(
				client,
				treasureCoordChan,
				treasureCoordChanUrgent,
				cashierChan,
				cashierChanUrgent,
				m.licensor,
				showStat,
			),
		)
	}

	m.explorer = NewExplorer(client, treasureCoordChan, treasureCoordChanUrgent, explorersCount, showStat)

	for i := 0; i < cashiersCount; i++ {
		m.cashiers = append(m.cashiers, NewCashier(client, cashierChan, cashierChanUrgent, showStat))
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

		time.Sleep(1 * time.Millisecond)
	}

	fmt.Println("healthCheck passed")
}

func (miner *Miner) Start() error {
	miner.licensor.Init()

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

	miner.explorer.Init()

	miner.healthCheck()
	go miner.explorer.Start(&wg)

	miner.licensor.Start()

	if miner.showStat {
		go miner.printStat()
	}

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
			DiggerStat.printStat(sub)
			CashierStat.printStat(sub)
		}
	}
}
