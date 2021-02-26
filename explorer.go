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

				sort.Slice(reportTreesSortedByDensity, func(i, j int) bool {
					return reportTreesSortedByDensity[i].Parent.Density > reportTreesSortedByDensity[j].Parent.Density
				})
			case outChan <- reportTreesSortedByDensity[0]:
				reportTreesSortedByDensity = reportTreesSortedByDensity[1:]
			case <-sortChan:
				sort.Slice(reportTreesSortedByDensity, func(i, j int) bool {
					return reportTreesSortedByDensity[i].Parent.Density > reportTreesSortedByDensity[j].Parent.Density
				})
			case reCalcTree := <-reCalcTreeChan:
				tree := reCalcTree
				for tree != nil {
					tree.setAmount(tree.Report.Amount - reCalcTree.Report.Amount)

					tree = tree.Parent
				}
				sort.Slice(reportTreesSortedByDensity, func(i, j int) bool {
					return reportTreesSortedByDensity[i].Parent.Density > reportTreesSortedByDensity[j].Parent.Density
				})
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
				if densestTree.Parent != nil && len(densestTree.Parent.Children) == 2 {
					for _, child := range densestTree.Parent.Children {
						if densestTree.Report != child.Report {
							child.setAmount(densestTree.Parent.Report.Amount - densestTree.Report.Amount)
							break
						}
					}
				}

				if densestTree.Parent != nil {
					for _, parentChild := range densestTree.Parent.Children {
						if parentChild.Density >= 1 {
							// send to digger chan
							e.treasureReportChan <- parentChild.Report

							reCalcTreeChan <- parentChild

							continue
						}

						if parentChild.Density > 0 {
							// set areas
							if parentChild.Report.Area.SizeX >= parentChild.Report.Area.SizeY {
								parentChild.Children = append(parentChild.Children, &ReportTree{
									Report: openapi.Report{
										Area: openapi.Area{
											PosX:  parentChild.Report.Area.PosX,
											PosY:  parentChild.Report.Area.PosY,
											SizeX: parentChild.Report.Area.SizeX/2 + parentChild.Report.Area.SizeX%2,
											SizeY: parentChild.Report.Area.SizeY,
										},
									},
									Parent: parentChild,
								})
								parentChild.Children = append(parentChild.Children, &ReportTree{
									Report: openapi.Report{
										Area: openapi.Area{
											PosX:  parentChild.Report.Area.PosX + parentChild.Children[0].Report.Area.SizeX,
											PosY:  parentChild.Report.Area.PosY,
											SizeX: parentChild.Report.Area.SizeX - parentChild.Children[0].Report.Area.SizeX,
											SizeY: parentChild.Report.Area.SizeY,
										},
									},
									Parent: parentChild,
								})
							} else {
								parentChild.Children = append(parentChild.Children, &ReportTree{
									Report: openapi.Report{
										Area: openapi.Area{
											PosX:  parentChild.Report.Area.PosX,
											PosY:  parentChild.Report.Area.PosY + parentChild.Report.Area.SizeY/2,
											SizeX: parentChild.Report.Area.SizeX,
											SizeY: parentChild.Report.Area.SizeY/2 + parentChild.Report.Area.SizeY%2,
										},
									},
									Parent: parentChild,
								})

								parentChild.Children = append(parentChild.Children, &ReportTree{
									Report: openapi.Report{
										Area: openapi.Area{
											PosX:  parentChild.Report.Area.PosX,
											PosY:  parentChild.Report.Area.PosY,
											SizeX: parentChild.Report.Area.SizeX,
											SizeY: parentChild.Report.Area.SizeY - parentChild.Children[0].Report.Area.SizeY,
										},
									},
									Parent: parentChild,
								})
							}

							inChan <- parentChild.Children[0]
						}
					}
				}
			}
		}(
			inChan,
			sortChan,
			outChan,
			reCalcTreeChan,
		)
	}
}
