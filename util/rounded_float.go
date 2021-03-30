package util

import "strconv"

type RoundedFloat float64

func (r RoundedFloat) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(r), 'f', 2, 32)), nil
}

func UpperPowerOfTwo(number int32) int32 {
	number--
	number |= number >> 1
	number |= number >> 2
	number |= number >> 4
	number |= number >> 8
	number |= number >> 16
	number++

	return number
}

func LowerPowerOfTwo(number int32) int32 {
	upperPowerOfTwo := UpperPowerOfTwo(number)
	if upperPowerOfTwo == number {
		return number
	}

	return upperPowerOfTwo / 2
}

func ClosestToPowerOfTwo(x, y, n int32) (int32, int32) {
	x1, y1 := x, y
	x2, y2 := x, y

	for x1*y1 > 2 && x1*y1 >= n {
		x1--
	}

	for x2*y2 > 2 && x2*y2 >= n {
		y2--
	}

	if x1*y1 > x2*y2 {
		return x1, y1
	}

	return x2, y2
}
