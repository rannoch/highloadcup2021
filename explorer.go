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
	defer wg.Done()

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

	var reportTreeSortedByDensity []*ReportTree

	for _, child := range rootReportTree.Children {
		for {
			report, respCode, _ := e.client.ExploreArea(child.Report.Area)
			if respCode == 200 {
				child.setReport(report)

				child.Parent.setAmount(child.Parent.Report.Amount + child.Report.Amount)
				reportTreeSortedByDensity = append(reportTreeSortedByDensity, child)

				break
			}
		}
	}

	for len(reportTreeSortedByDensity) > 0 {
		densestTree := reportTreeSortedByDensity[0]
		reportTreeSortedByDensity = reportTreeSortedByDensity[1:]

		if densestTree.Density >= 1 {
			// recalculate reportTreeSortedByDensity recursively
			tree := densestTree.Parent
			for tree != nil {
				tree.setAmount(tree.Report.Amount - densestTree.Report.Amount)

				tree = tree.Parent
			}

			sort.Slice(reportTreeSortedByDensity, func(i, j int) bool {
				return reportTreeSortedByDensity[i].Density > reportTreeSortedByDensity[j].Density
			})

			// send to digger chan
			e.treasureReportChan <- densestTree.Report

			continue
		}

		// split by two, create left child and right child
		densestTree.Children = append(densestTree.Children, &ReportTree{
			Parent: densestTree,
		})

		densestTree.Children = append(densestTree.Children, &ReportTree{
			Parent: densestTree,
		})

		// set areas
		if densestTree.Report.Area.SizeX >= densestTree.Report.Area.SizeY {
			densestTree.Children[0].Report = openapi.Report{
				Area: openapi.Area{
					PosX:  densestTree.Report.Area.PosX,
					PosY:  densestTree.Report.Area.PosY,
					SizeX: densestTree.Report.Area.SizeX/2 + densestTree.Report.Area.SizeX%2,
					SizeY: densestTree.Report.Area.SizeY,
				},
			}

			densestTree.Children[1].Report = openapi.Report{
				Area: openapi.Area{
					PosX:  densestTree.Report.Area.PosX + densestTree.Children[0].Report.Area.SizeX,
					PosY:  densestTree.Report.Area.PosY,
					SizeX: densestTree.Report.Area.SizeX - densestTree.Children[0].Report.Area.SizeX,
					SizeY: densestTree.Report.Area.SizeY,
				},
			}
		} else {
			densestTree.Children[0].Report = openapi.Report{
				Area: openapi.Area{
					PosX:  densestTree.Report.Area.PosX,
					PosY:  densestTree.Report.Area.PosY + densestTree.Report.Area.SizeY/2,
					SizeX: densestTree.Report.Area.SizeX,
					SizeY: densestTree.Report.Area.SizeY/2 + densestTree.Report.Area.SizeY%2,
				},
			}
			densestTree.Children[1].Report = openapi.Report{
				Area: openapi.Area{
					PosX:  densestTree.Report.Area.PosX,
					PosY:  densestTree.Report.Area.PosY,
					SizeX: densestTree.Report.Area.SizeX,
					SizeY: densestTree.Report.Area.SizeY - densestTree.Children[0].Report.Area.SizeY,
				},
			}
		}

		// explore left and calculate right amount
		for {
			report, respCode, _ := e.client.ExploreArea(densestTree.Children[0].Report.Area)
			if respCode == 200 {
				densestTree.Children[0].setReport(report)

				densestTree.Children[1].setAmount(densestTree.Report.Amount - densestTree.Children[0].Report.Amount)
				break
			}
		}

		needToSort := false

		for _, child := range densestTree.Children {
			if child.Density > 0 {
				reportTreeSortedByDensity = append(reportTreeSortedByDensity, child)
				needToSort = true
			}
		}

		if needToSort {
			sort.Slice(reportTreeSortedByDensity, func(i, j int) bool {
				return reportTreeSortedByDensity[i].Density > reportTreeSortedByDensity[j].Density
			})
		}
	}
}
