package miner

import (
	"encoding/json"
	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/rannoch/highloadcup2021/api_client"
	"github.com/rannoch/highloadcup2021/miner/model"
	"math"
	"sync"
	"time"
)

type Explorer struct {
	client *api_client.Client

	treasureReportChan      chan<- model.Report
	treasureCoordChanUrgent chan<- model.Report

	reportTreesSortedByDensity *RedBlackTreeExtended

	inChan  chan *ReportTree
	outChan chan *ReportTree

	workerCount int

	explorerStat explorerStat

	showStat bool
}

type explorerStat struct {
	requestsTotal       int64
	requestTimeByArea   map[int32]time.Duration
	requestCountByArea  map[int32]int64
	diggerWaitTimeTotal time.Duration

	requestTimeMutex sync.RWMutex
}

func NewExplorer(
	client *api_client.Client,
	treasureReportChan, treasureCoordChanUrgent chan<- model.Report,
	workerCount int,
	showStat bool,
) *Explorer {
	e := &Explorer{client: client, treasureReportChan: treasureReportChan, workerCount: workerCount}
	e.treasureCoordChanUrgent = treasureCoordChanUrgent

	e.explorerStat.requestTimeByArea = make(map[int32]time.Duration)
	e.explorerStat.requestCountByArea = make(map[int32]int64)

	e.inChan = make(chan *ReportTree, 1000)
	e.outChan = make(chan *ReportTree, 0)

	e.showStat = showStat

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

func (e *Explorer) Init() {
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

	e.reportTreesSortedByDensity = NewRedBlackTreeExtended(reportTreeComparator)

	for _, child := range rootReportTree.Children {
		e.reportTreesSortedByDensity.Put(child, child)
	}
}

func (e *Explorer) Start(wg *sync.WaitGroup) {
	go e.reportTreeSorter()

	wg.Add(e.workerCount)

	for i := 0; i < e.workerCount; i++ {
		go e.explore(wg)
	}
}

func (e *Explorer) reportTreeSorter() {
	for {
		select {
		case reportTree := <-e.inChan:
			e.reportTreesSortedByDensity.Put(reportTree, reportTree)
		case e.outChan <- e.reportTreesSortedByDensity.GetMaxNodeValue():
			e.reportTreesSortedByDensity.RemoveMax()
		}
	}
}

func (e *Explorer) PrintStat(duration time.Duration) {
	e.explorerStat.requestTimeMutex.RLock()
	requestTimeString, _ := json.Marshal(e.explorerStat.requestTimeByArea)

	println("Explores total after "+duration.String(), e.explorerStat.requestsTotal)

	println("Explore requests time by area stat after " + duration.String())
	println(string(requestTimeString))

	requestCountString, _ := json.Marshal(e.explorerStat.requestCountByArea)
	println("Explore requests count by area stat after " + duration.String())
	println(string(requestCountString))

	var requestAvgTime = make(map[int32]float64, len(e.explorerStat.requestTimeByArea))

	for area := range e.explorerStat.requestTimeByArea {
		requestAvgTime[area] = math.Round(float64(e.explorerStat.requestTimeByArea[area])/float64(e.explorerStat.requestCountByArea[area])/float64(time.Second)*1000) / 1000
	}

	requestAvgTimeString, _ := json.Marshal(requestAvgTime)
	println("Explore requests avg time by area stat after " + duration.String())
	println(string(requestAvgTimeString))

	println("Explore digger wait time total " + e.explorerStat.diggerWaitTimeTotal.String())
	println()

	e.explorerStat.requestTimeMutex.RUnlock()
}

func (e *Explorer) explore(
	wg *sync.WaitGroup,
) {
	wg.Done()

	for densestTree := range e.outChan {
		if densestTree == nil {
			continue
		}

		// explore left and calculate neighbor amount
		for {
			report, respCode, requestTime, _ := e.client.ExploreArea(densestTree.Report.Area)

			// stat
			if e.showStat {
				e.explorerStat.requestTimeMutex.Lock()
				e.explorerStat.requestsTotal++
				e.explorerStat.requestTimeByArea[densestTree.Report.Area.SizeX*densestTree.Report.Area.SizeY] += requestTime
				e.explorerStat.requestCountByArea[densestTree.Report.Area.SizeX*densestTree.Report.Area.SizeY]++
				e.explorerStat.requestTimeMutex.Unlock()
			}

			if respCode == 200 {
				densestTree.setReport(report)
				break
			}
		}

		// update neighbour amount
		if densestTree.Neighbour != nil {
			densestTree.Neighbour.setAmount(densestTree.Parent.Report.Amount - densestTree.Report.Amount)
		}

		e.processTree(densestTree)
		e.processTree(densestTree.Neighbour)
	}
}

func (e *Explorer) processTree(
	tree *ReportTree,
) {
	if tree == nil {
		return
	}

	var sendingToDiggerStartTime time.Time

	if tree.Density >= 1 && tree.AreaSize == 1 {
		// send to digger chan

		treasureChan := e.treasureReportChan
		if tree.Density > 1 {
			treasureChan = e.treasureCoordChanUrgent
		}

		if e.showStat {
			sendingToDiggerStartTime = time.Now()
		}
		select {
		case treasureChan <- tree.Report:
			if e.showStat {
				e.explorerStat.requestTimeMutex.Lock()
				e.explorerStat.diggerWaitTimeTotal += time.Now().Sub(sendingToDiggerStartTime)
				e.explorerStat.requestTimeMutex.Unlock()
			}
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

		e.inChan <- tree.Children[0]
	}
}
