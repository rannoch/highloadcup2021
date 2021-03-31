package model

import "math"

func (r Report) Density() float32 {
	if r.Area.Size() == 0 {
		return 0
	}

	return float32(r.Amount) / float32(r.Area.Size())
}

func (a Area) Size() int32 {
	return a.SizeX * a.SizeY
}

func (a Area) Empty() bool {
	return a.SizeX == 0
}

func (a Area) ExploreCost() int32 {
	if a.Size() <= 3 {
		return 1
	}

	return int32(math.Log2(float64(a.Size())))
}

type ExploreArea struct {
	Index int64

	ParentReport Report

	AreaSection1 Area
	AreaSection2 Area
}
