package main

import (
	"fmt"
	openapi "github.com/rannoch/highloadcup2021/client"
	"sync"
	"time"
)

type Miner struct {
	balance openapi.Balance

	explorer *Explorer
	diggers  []*Digger

	cashierChan chan string

	client *Client
}

func NewMiner(client *Client, diggersCount int) *Miner {
	m := &Miner{client: client}

	var treasureCoordChan = make(chan openapi.Report, 100)
	m.cashierChan = make(chan string, 1000)

	for i := 0; i < diggersCount; i++ {
		m.diggers = append(m.diggers, NewDigger(client, treasureCoordChan, m.cashierChan))
	}

	m.explorer = NewExplorer(client, treasureCoordChan)

	return m
}

func (miner *Miner) cashier(c <-chan string) {
	for {
		select {
		case treasure := <-c:
			for {
				_, _, err := miner.client.Cash(fmt.Sprintf("\"%s\"", treasure))
				if err == nil {
					break
				}
			}
		}
	}
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
		go digger.Start()
	}

	wg.Wait()

	return nil
}
