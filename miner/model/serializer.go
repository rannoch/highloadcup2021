package model

import (
	"strconv"
)

func (d Dig) MarshalJSON() ([]byte, error) {
	return []byte(`{"licenseID":` + strconv.Itoa(int(d.LicenseID)) + `,"posX":` + strconv.Itoa(int(d.PosX)) + `,"posY":` + strconv.Itoa(int(d.PosY)) + `,"depth":` + strconv.Itoa(int(d.Depth)) + `}`), nil
}

func (a Area) MarshalJSON() ([]byte, error) {
	return []byte(`{"posX":` + strconv.Itoa(int(a.PosX)) + `,"posY":` + strconv.Itoa(int(a.PosY)) + `,"sizeX":` + strconv.Itoa(int(a.SizeX)) + `,"sizeY":` + strconv.Itoa(int(a.SizeY)) + `}`), nil
}
