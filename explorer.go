package main

import (
	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/rannoch/highloadcup2021/model"
	"math"
	"sync"
	"time"
)

type Explorer struct {
	client *Client

	treasureReportChan chan<- model.Report

	workerCount int

	requestsTotal       int64
	requestTimeByArea   map[int32]time.Duration
	requestCountByArea  map[int32]int64
	diggerWaitTimeTotal time.Duration

	requestTimeMutex sync.RWMutex
}

func NewExplorer(client *Client, treasureReportChan chan<- model.Report, workerCount int) *Explorer {
	e := &Explorer{client: client, treasureReportChan: treasureReportChan, workerCount: workerCount}
	e.requestTimeByArea = make(map[int32]time.Duration)
	e.requestCountByArea = make(map[int32]int64)

	return e
}

func (r *ReportTree) setReport(report model.Report) {
	r.Report = report

	r.calculateDensity()
	r.AreaSize = report.Area.SizeX * report.Area.SizeY
}

func (r *ReportTree) setAmount(amount int32) {
	r.Report.Amount = amount

	r.calculateDensity()
}

func (r *ReportTree) calculateDensity() {
	if r.Report.Area.SizeX*r.Report.Area.SizeY == 0 {
		r.Density = 0
		return
	}

	r.Density = float32(r.Report.Amount) / float32(r.Report.Area.SizeX*r.Report.Area.SizeY)
}

type ReportTree struct {
	Report model.Report

	Density float32

	AreaSize int32

	Parent *ReportTree

	Children []*ReportTree

	Neighbour *ReportTree
}

var reportTreeComparator = rbt.NewWith(func(a, b interface{}) int {
	if b == nil {
		return 1
	}

	reportTree1 := a.(*ReportTree)
	reportTree2 := b.(*ReportTree)

	switch true {
	case reportTree1.Report == reportTree2.Report:
		return 0
	case reportTree1.Parent.Density == 0 || reportTree1.Parent.Density > reportTree2.Parent.Density:
		return 1
	case reportTree1.Parent.Density < reportTree2.Parent.Density:
		return -1
	}

	return 1
})

func (e *Explorer) Start(wg *sync.WaitGroup) {
	rootReportTree := &ReportTree{
		Report: model.Report{
			Area: model.Area{
				PosX:  0,
				PosY:  0,
				SizeX: 3500,
				SizeY: 3500,
			},
			Amount: 0,
		},
		Density: 0,

		Parent: nil,
	}

	const xStep = 350
	const yStep = 350

	// calculate initial
	for i := 0; i < 25; i++ {
		area := model.Area{
			PosX:  int32(i%5) * xStep,
			PosY:  int32(i/5) * yStep,
			SizeX: xStep,
			SizeY: yStep,
		}

		rootReportTree.Children = append(rootReportTree.Children, &ReportTree{
			Report: model.Report{
				Area: area,
			},
			Parent: rootReportTree,
		})
	}

	var reportTreesSortedByDensity = NewRedBlackTreeExtended(reportTreeComparator)

	for _, child := range rootReportTree.Children {
		reportTreesSortedByDensity.Put(child, child)
	}

	var inChan = make(chan *ReportTree, 1000)
	var outChan = make(chan *ReportTree, 0)

	go e.reportTreeSorter(reportTreesSortedByDensity, inChan, outChan)

	wg.Add(e.workerCount)

	for i := 0; i < e.workerCount; i++ {
		go e.explore(inChan, outChan, wg)
	}
}

func (e *Explorer) reportTreeSorter(
	reportTreesSortedByDensity *RedBlackTreeExtended,
	inChan <-chan *ReportTree,
	outChan chan<- *ReportTree,
) {
	for {
		select {
		case reportTree := <-inChan:
			reportTreesSortedByDensity.Put(reportTree, reportTree)
		case outChan <- reportTreesSortedByDensity.GetMaxNodeValue():
			reportTreesSortedByDensity.RemoveMax()
		}
	}
}

func (e *Explorer) PrintStat(duration time.Duration) {
	e.requestTimeMutex.RLock()
	requestTimeString, _ := json.Marshal(e.requestTimeByArea)

	println("Explores total after " + duration.String())
	println(e.requestsTotal)

	println("Explore requests time by area stat after " + duration.String())
	println(string(requestTimeString))

	requestCountString, _ := json.Marshal(e.requestCountByArea)
	println("Explore requests count by area stat after " + duration.String())
	println(string(requestCountString))

	var requestAvgTime = make(map[int32]float64, len(e.requestTimeByArea))

	for area := range e.requestTimeByArea {
		requestAvgTime[area] = math.Round(float64(e.requestTimeByArea[area])/float64(e.requestCountByArea[area])/float64(time.Second)*1000) / 1000
	}

	requestAvgTimeString, _ := json.Marshal(requestAvgTime)
	println("Explore requests avg time by area stat after " + duration.String())
	println(string(requestAvgTimeString))

	println("Explore digger wait time total " + e.diggerWaitTimeTotal.String())
	println()

	e.requestTimeMutex.RUnlock()
}

func (e *Explorer) explore(
	inChan chan<- *ReportTree,
	outChan <-chan *ReportTree,
	wg *sync.WaitGroup,
) {
	wg.Done()

	for densestTree := range outChan {
		// explore left and calculate neighbor amount
		for {
			report, respCode, requestTime, _ := e.client.ExploreArea(densestTree.Report.Area)

			// stat
			e.requestTimeMutex.Lock()
			e.requestsTotal++
			e.requestTimeByArea[densestTree.Report.Area.SizeX*densestTree.Report.Area.SizeY] += requestTime
			e.requestCountByArea[densestTree.Report.Area.SizeX*densestTree.Report.Area.SizeY]++
			e.requestTimeMutex.Unlock()

			if respCode == 200 {
				densestTree.setReport(report)
				break
			}
		}

		// update neighbour amount
		if densestTree.Neighbour != nil {
			densestTree.Neighbour.setAmount(densestTree.Parent.Report.Amount - densestTree.Report.Amount)
		}

		e.processTree(densestTree, inChan)
		e.processTree(densestTree.Neighbour, inChan)
	}
}

func (e *Explorer) processTree(
	tree *ReportTree,
	inChan chan<- *ReportTree,
) {
	if tree == nil {
		return
	}

	if tree.Density >= 1 && tree.AreaSize == 1 {
		// send to digger chan
		sendingToDiggerStartTime := time.Now()
		select {
		case e.treasureReportChan <- tree.Report:
			e.requestTimeMutex.Lock()
			e.diggerWaitTimeTotal += time.Now().Sub(sendingToDiggerStartTime)
			e.requestTimeMutex.Unlock()
		}

		return
	}

	if tree.Density > 0 {
		// set areas
		if tree.Report.Area.SizeX >= tree.Report.Area.SizeY {
			tree.Children = append(tree.Children, &ReportTree{
				Report: model.Report{
					Area: model.Area{
						PosX:  tree.Report.Area.PosX,
						PosY:  tree.Report.Area.PosY,
						SizeX: tree.Report.Area.SizeX/2 + tree.Report.Area.SizeX%2,
						SizeY: tree.Report.Area.SizeY,
					},
				},
				Parent: tree,
			})
			tree.Children = append(tree.Children, &ReportTree{
				Report: model.Report{
					Area: model.Area{
						PosX:  tree.Report.Area.PosX + tree.Children[0].Report.Area.SizeX,
						PosY:  tree.Report.Area.PosY,
						SizeX: tree.Report.Area.SizeX - tree.Children[0].Report.Area.SizeX,
						SizeY: tree.Report.Area.SizeY,
					},
				},
				Parent: tree,
			})
		} else {
			tree.Children = append(tree.Children, &ReportTree{
				Report: model.Report{
					Area: model.Area{
						PosX:  tree.Report.Area.PosX,
						PosY:  tree.Report.Area.PosY + tree.Report.Area.SizeY/2,
						SizeX: tree.Report.Area.SizeX,
						SizeY: tree.Report.Area.SizeY/2 + tree.Report.Area.SizeY%2,
					},
				},
				Parent: tree,
			})

			tree.Children = append(tree.Children, &ReportTree{
				Report: model.Report{
					Area: model.Area{
						PosX:  tree.Report.Area.PosX,
						PosY:  tree.Report.Area.PosY,
						SizeX: tree.Report.Area.SizeX,
						SizeY: tree.Report.Area.SizeY - tree.Children[0].Report.Area.SizeY,
					},
				},
				Parent: tree,
			})
		}

		tree.Children[0].Neighbour = tree.Children[1]
		tree.Children[1].Neighbour = tree.Children[0]

		inChan <- tree.Children[0]
	}
}
