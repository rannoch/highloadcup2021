package miner

import (
	"container/heap"
	"encoding/json"
	"github.com/rannoch/highloadcup2021/api_client"
	"github.com/rannoch/highloadcup2021/miner/model"
	"github.com/rannoch/highloadcup2021/util"
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

	e.priorityQueue = make([]model.ExploreArea, 0, 10000)

	heap.Init(&e.priorityQueue)

	e.showStat = showStat

	return e
}

func (e *Explorer) Init() {
	const xPart int32 = 5
	const yPart int32 = 5
	const xSize = 3500
	const ySize = 3500

	var xStep = xSize / xPart
	var yStep = ySize / xPart

	var i int32
	// calculate initial
	for i = 0; i < xPart*yPart; i++ {
		area := model.Area{
			PosX:  i % xPart * xStep,
			PosY:  i / yPart * yStep,
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

	for {
		e.priorityQueueMutex.Lock()
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

func (e *Explorer) processReport(
	report model.Report,
) {
	var sendingToDiggerStartTime time.Time

	if report.Density() >= 1 && report.Area.Size() == 1 {
		// send to digger chan

		treasureChan := e.treasureReportChan
		if report.Density() > 1 {
			treasureChan = e.treasureCoordChanUrgent
		}

		if e.showStat {
			sendingToDiggerStartTime = time.Now()
		}
		select {
		case treasureChan <- report:
			if e.showStat {
				e.explorerStat.requestTimeMutex.Lock()
				e.explorerStat.diggerWaitTimeTotal += time.Now().Sub(sendingToDiggerStartTime)
				e.explorerStat.requestTimeMutex.Unlock()
			}
		}

		return
	}

	if report.Density() > 0 {
		exploreArea := model.ExploreArea{
			Index:        atomic.LoadInt64(&e.priorityQueueIndex),
			ParentReport: report,
		}

		atomic.AddInt64(&e.priorityQueueIndex, 1)

		areaLowerPowerOfTwoSize := util.LowerPowerOfTwo(report.Area.Size() / 2)

		// set areas
		if report.Area.SizeX >= report.Area.SizeY {
			exploreArea.AreaSection1 = model.Area{
				PosX:  report.Area.PosX,
				PosY:  report.Area.PosY,
				SizeX: report.Area.SizeX/2 + report.Area.SizeX%2,
				SizeY: report.Area.SizeY,
			}

			for exploreArea.AreaSection1.SizeX > 1 &&
				exploreArea.AreaSection1.Size() > 2 &&
				exploreArea.AreaSection1.Size() > areaLowerPowerOfTwoSize {

				exploreArea.AreaSection1.SizeX--
			}

			exploreArea.AreaSection2 = model.Area{
				PosX:  report.Area.PosX + exploreArea.AreaSection1.SizeX,
				PosY:  report.Area.PosY,
				SizeX: report.Area.SizeX - exploreArea.AreaSection1.SizeX,
				SizeY: report.Area.SizeY,
			}
		} else {
			exploreArea.AreaSection1 = model.Area{
				PosX:  report.Area.PosX,
				PosY:  report.Area.PosY,
				SizeX: report.Area.SizeX,
				SizeY: report.Area.SizeY/2 + report.Area.SizeY%2,
			}

			for exploreArea.AreaSection1.SizeY > 1 &&
				exploreArea.AreaSection1.Size() > 2 &&
				exploreArea.AreaSection1.Size() > areaLowerPowerOfTwoSize {
				exploreArea.AreaSection1.SizeY--
			}

			exploreArea.AreaSection2 = model.Area{
				PosX:  report.Area.PosX,
				PosY:  report.Area.PosY + exploreArea.AreaSection1.SizeY,
				SizeX: report.Area.SizeX,
				SizeY: report.Area.SizeY - exploreArea.AreaSection1.SizeY,
			}
		}

		e.priorityQueueMutex.Lock()
		heap.Push(
			&e.priorityQueue, exploreArea,
		)
		e.priorityQueueMutex.Unlock()
	}
}
