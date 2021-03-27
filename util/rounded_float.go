package util

import "strconv"

type RoundedFloat float64

func (r RoundedFloat) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(r), 'f', 2, 32)), nil
}
