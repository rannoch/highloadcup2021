package main

import (
	openapi "github.com/rannoch/highloadcup2021/client"
	"sort"
	"sync"
)

type Explorer struct {
	client *Client

	treasureReportChan chan<- openapi.Report
}

func NewExplorer(client *Client, treasureCoordChan chan<- openapi.Report) *Explorer {
	return &Explorer{client: client, treasureReportChan: treasureCoordChan}
}

func (r *ReportTree) setReport(report openapi.Report) {
	r.Report = report

	r.calculateDensity()
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
	Report openapi.Report

	Density float32

	Parent *ReportTree

	Children []*ReportTree

	Neighbour *ReportTree
}

func (e *Explorer) Start(wg *sync.WaitGroup) {
	rootReportTree := &ReportTree{
		Report: openapi.Report{
			Area: openapi.Area{
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

	const xStep = 1750
	const yStep = 350

	// calculate initial
	for i := 0; i < 20; i++ {
		area := openapi.Area{
			PosX:  int32(i%2) * xStep,
			PosY:  int32(i/2) * yStep,
			SizeX: xStep,
			SizeY: yStep,
		}

		rootReportTree.Children = append(rootReportTree.Children, &ReportTree{
			Report: openapi.Report{
				Area: area,
			},
			Parent: rootReportTree,
		})
	}

	var inChan = make(chan *ReportTree, 1000)
	var outChan = make(chan *ReportTree)

	var sortChan = make(chan interface{})
	var reCalcTreeChan = make(chan *ReportTree, 1000)

	go func(
		inChan <-chan *ReportTree,
		sortChan <-chan interface{},
		outChan chan<- *ReportTree,
		reCalcTreeChan <-chan *ReportTree,
	) {
		var reportTreesSortedByDensity []*ReportTree
		reportTreesSortedByDensity = append(reportTreesSortedByDensity, rootReportTree.Children...)

		for {
			select {
			case reportTree := <-inChan:
				reportTreesSortedByDensity = append(reportTreesSortedByDensity, reportTree)

				sortTree(reportTreesSortedByDensity)
			case outChan <- reportTreesSortedByDensity[0]:
				reportTreesSortedByDensity = reportTreesSortedByDensity[1:]
			case <-sortChan:
				sortTree(reportTreesSortedByDensity)
			case reCalcTree := <-reCalcTreeChan:
				tree := reCalcTree
				for tree != nil {
					tree.setAmount(tree.Report.Amount - reCalcTree.Report.Amount)

					tree = tree.Parent
				}
				sortTree(reportTreesSortedByDensity)
			}
		}
	}(inChan, sortChan, outChan, reCalcTreeChan)

	workersCount := 5

	wg.Add(workersCount)

	for i := 0; i < workersCount; i++ {
		go func(
			inChan chan<- *ReportTree,
			sortChan chan<- interface{},
			outChan <-chan *ReportTree,
			reCalcTreeChan chan<- *ReportTree,
		) {
			wg.Done()

			for densestTree := range outChan {
				// explore left and calculate neighbor amount
				for {
					report, respCode, _ := e.client.ExploreArea(densestTree.Report.Area)
					if respCode == 200 {
						densestTree.setReport(report)
						break
					}
				}

				// update neighbour amount
				if densestTree.Neighbour != nil {
					densestTree.Neighbour.setAmount(densestTree.Parent.Report.Amount - densestTree.Report.Amount)
				}

				e.processTree(densestTree, inChan, reCalcTreeChan)
				e.processTree(densestTree.Neighbour, inChan, reCalcTreeChan)
			}
		}(
			inChan,
			sortChan,
			outChan,
			reCalcTreeChan,
		)
	}
}

func sortTree(reportTreesSortedByDensity []*ReportTree) {
	sort.Slice(reportTreesSortedByDensity, func(i, j int) bool {
		if reportTreesSortedByDensity[i].Parent.Density == 0 {
			return true
		}

		return reportTreesSortedByDensity[i].Parent.Density > reportTreesSortedByDensity[j].Parent.Density
	})
}

func (e *Explorer) processTree(
	tree *ReportTree,
	inChan chan<- *ReportTree,
	reCalcTreeChan chan<- *ReportTree,
) {
	if tree == nil {
		return
	}

	if tree.Density >= 1 {
		// send to digger chan
		e.treasureReportChan <- tree.Report

		reCalcTreeChan <- tree
		return
	}

	if tree.Density > 0 {
		// set areas
		if tree.Report.Area.SizeX >= tree.Report.Area.SizeY {
			tree.Children = append(tree.Children, &ReportTree{
				Report: openapi.Report{
					Area: openapi.Area{
						PosX:  tree.Report.Area.PosX,
						PosY:  tree.Report.Area.PosY,
						SizeX: tree.Report.Area.SizeX/2 + tree.Report.Area.SizeX%2,
						SizeY: tree.Report.Area.SizeY,
					},
				},
				Parent: tree,
			})
			tree.Children = append(tree.Children, &ReportTree{
				Report: openapi.Report{
					Area: openapi.Area{
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
				Report: openapi.Report{
					Area: openapi.Area{
						PosX:  tree.Report.Area.PosX,
						PosY:  tree.Report.Area.PosY + tree.Report.Area.SizeY/2,
						SizeX: tree.Report.Area.SizeX,
						SizeY: tree.Report.Area.SizeY/2 + tree.Report.Area.SizeY%2,
					},
				},
				Parent: tree,
			})

			tree.Children = append(tree.Children, &ReportTree{
				Report: openapi.Report{
					Area: openapi.Area{
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