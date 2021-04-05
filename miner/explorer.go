package miner

import (
	"container/heap"
	"encoding/json"
	"github.com/rannoch/highloadcup2021/api_client"
	"github.com/rannoch/highloadcup2021/miner/model"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type Explorer struct {
	client *api_client.Client

	treasureReportChan      chan<- model.Report
	treasureCoordChanUrgent chan<- model.Report

	priorityQueue      PriorityQueue
	priorityQueueIndex int64
	priorityQueueMutex sync.RWMutex

	workerCount int

	explorerStat explorerStat

	showStat bool
}

type PriorityQueue []model.ExploreArea

func (p PriorityQueue) Len() int {
	return len(p)
}

func (p PriorityQueue) Less(i, j int) bool {
	return p[i].ParentReport.Density() == 0 || p[i].ParentReport.Density() > p[j].ParentReport.Density()
}

func (p PriorityQueue) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].Index = int64(i)
	p[j].Index = int64(j)
}

func (p *PriorityQueue) Push(x interface{}) {
	n := len(*p)
	item := x.(model.ExploreArea)
	item.Index = int64(n)
	*p = append(*p, item)
}

func (p *PriorityQueue) Pop() interface{} {
	old := *p
	n := len(old)
	item := old[n-1]
	//old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*p = old[0 : n-1]
	return item
}

type explorerStat struct {
	requestsTotal      int64
	requestTimeByArea  map[int32]time.Duration
	requestCountByArea map[int32]int64

	responseCodes map[int]int

	treasuresTotal      int64
	treasureDoubleTotal map[int32]int32

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
	e.explorerStat.responseCodes = make(map[int]int)
	e.explorerStat.treasureDoubleTotal = make(map[int32]int32)

	e.priorityQueue = make([]model.ExploreArea, 0, 10000)

	heap.Init(&e.priorityQueue)

	e.showStat = showStat

	return e
}

func (e *Explorer) Init() {
	const xPart int32 = 4
	const yPart int32 = 4
	const xSize = 3072
	const ySize = 2560

	var xStep = xSize / xPart
	var yStep = ySize / yPart

	var i int32
	// calculate initial
	for i = 0; i < xPart*yPart; i++ {
		area := model.Area{
			PosX:  i % xPart * xStep,
			PosY:  i / xPart * yStep,
			SizeX: xStep,
			SizeY: yStep,
		}

		heap.Push(
			&e.priorityQueue, model.ExploreArea{
				Index:        e.priorityQueueIndex,
				AreaSection1: area,
			},
		)

		e.priorityQueueIndex++
	}
}

func (e *Explorer) Start(wg *sync.WaitGroup) {
	wg.Add(e.workerCount)

	for i := 0; i < e.workerCount; i++ {
		go e.explore(wg)
	}
}

func (e *Explorer) PrintStat(duration time.Duration) {
	e.explorerStat.requestTimeMutex.RLock()

	println("Explores total after "+duration.String(), e.explorerStat.requestsTotal, ",treasure cells found -", e.explorerStat.treasuresTotal)
	responseCodesJson, _ := json.Marshal(e.explorerStat.responseCodes)
	println("Explore response codes: " + string(responseCodesJson))

	treasureDoubleTotalJson, _ := json.Marshal(e.explorerStat.treasureDoubleTotal)
	println("Double treasures: " + string(treasureDoubleTotalJson))

	requestTimeString, _ := json.Marshal(e.explorerStat.requestTimeByArea)
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

	for {
		e.priorityQueueMutex.Lock()
		if e.priorityQueue.Len() == 0 {
			e.priorityQueueMutex.Unlock()
			continue
		}

		exploreArea := heap.Pop(&e.priorityQueue).(model.ExploreArea)
		e.priorityQueueMutex.Unlock()

		// explore left and calculate neighbor amount
		for {
			report, respCode, requestTime, _ := e.client.ExploreArea(exploreArea.AreaSection1)

			// stat
			if e.showStat {
				e.explorerStat.requestTimeMutex.Lock()
				e.explorerStat.requestsTotal++
				e.explorerStat.requestTimeByArea[exploreArea.AreaSection1.Size()] += requestTime
				e.explorerStat.requestCountByArea[exploreArea.AreaSection1.Size()]++
				e.explorerStat.responseCodes[respCode]++
				e.explorerStat.requestTimeMutex.Unlock()
			}

			if respCode == 200 {
				e.processReport(report)

				if !exploreArea.AreaSection2.Empty() {
					report2 := model.Report{
						Area:   exploreArea.AreaSection2,
						Amount: exploreArea.ParentReport.Amount - report.Amount,
					}

					e.processReport(report2)
				}

				break
			}
		}
	}
}

func (e *Explorer) sendReport(report model.Report) {
	var sendingToDiggerStartTime time.Time

	if e.showStat {
		sendingToDiggerStartTime = time.Now()
	}

	treasureReportChan := e.treasureReportChan
	if report.Density() > 1 {
		treasureReportChan = e.treasureCoordChanUrgent
	}

	select {
	case treasureReportChan <- report:
		if e.showStat {
			e.explorerStat.requestTimeMutex.Lock()
			e.explorerStat.diggerWaitTimeTotal += time.Now().Sub(sendingToDiggerStartTime)
			e.explorerStat.requestTimeMutex.Unlock()
		}
	}
}

func (e *Explorer) processReport(
	report model.Report,
) {
	if report.Density() >= 1 && report.Area.Size() == 1 {
		if e.showStat {
			e.explorerStat.requestTimeMutex.Lock()
			e.explorerStat.treasuresTotal++
			e.explorerStat.requestTimeMutex.Unlock()
		}

		if report.Density() > 1 {
			if e.showStat {
				e.explorerStat.requestTimeMutex.Lock()
				e.explorerStat.treasureDoubleTotal[int32(report.Density())]++
				e.explorerStat.requestTimeMutex.Unlock()
			}
		}

		e.sendReport(report)
		return
	}

	if report.Density() > 0 {
		exploreArea := model.ExploreArea{
			Index:        atomic.LoadInt64(&e.priorityQueueIndex),
			ParentReport: report,
		}

		atomic.AddInt64(&e.priorityQueueIndex, 1)

		variant1AreaSection1 := model.Area{
			PosX:  report.Area.PosX,
			PosY:  report.Area.PosY,
			SizeX: report.Area.SizeX / 2,
			SizeY: report.Area.SizeY,
		}

		variant1AreaSection2 := model.Area{
			PosX:  report.Area.PosX + variant1AreaSection1.SizeX,
			PosY:  report.Area.PosY,
			SizeX: report.Area.SizeX - variant1AreaSection1.SizeX,
			SizeY: report.Area.SizeY,
		}

		variant2AreaSection1 := model.Area{
			PosX:  report.Area.PosX,
			PosY:  report.Area.PosY,
			SizeX: report.Area.SizeX,
			SizeY: report.Area.SizeY / 2,
		}

		variant2AreaSection2 := model.Area{
			PosX:  report.Area.PosX,
			PosY:  report.Area.PosY + variant2AreaSection1.SizeY,
			SizeX: report.Area.SizeX,
			SizeY: report.Area.SizeY - variant2AreaSection1.SizeY,
		}

		if variant1AreaSection1.Size() == 0 || variant1AreaSection2.Size() == 0 {
			exploreArea.AreaSection1 = variant2AreaSection1
			exploreArea.AreaSection2 = variant2AreaSection2
		} else if variant2AreaSection1.Size() == 0 || variant2AreaSection2.Size() == 0 {
			exploreArea.AreaSection1 = variant1AreaSection1
			exploreArea.AreaSection2 = variant1AreaSection2
		} else if variant1AreaSection1.ExploreCost()+variant1AreaSection2.ExploreCost() == variant2AreaSection1.ExploreCost()+variant2AreaSection2.ExploreCost() {
			if math.Abs(float64(variant1AreaSection1.ExploreCost()-variant1AreaSection2.ExploreCost())) < math.Abs(float64(variant2AreaSection1.ExploreCost()-variant2AreaSection2.ExploreCost())) {
				exploreArea.AreaSection1 = variant1AreaSection1
				exploreArea.AreaSection2 = variant1AreaSection2
			} else {
				exploreArea.AreaSection1 = variant2AreaSection1
				exploreArea.AreaSection2 = variant2AreaSection2
			}
		} else if variant1AreaSection1.ExploreCost()+variant1AreaSection2.ExploreCost() < variant2AreaSection1.ExploreCost()+variant2AreaSection2.ExploreCost() {
			exploreArea.AreaSection1 = variant1AreaSection1
			exploreArea.AreaSection2 = variant1AreaSection2
		} else {
			exploreArea.AreaSection1 = variant2AreaSection1
			exploreArea.AreaSection2 = variant2AreaSection2
		}

		e.priorityQueueMutex.Lock()
		heap.Push(
			&e.priorityQueue, exploreArea,
		)
		e.priorityQueueMutex.Unlock()
	}
}
