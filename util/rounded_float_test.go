package util

import (
	"testing"
)

func TestClosestToTwo(t *testing.T) {
	type args struct {
		number int32
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "10",
			args: args{10},
			want: 16,
		},
		{
			name: "15",
			args: args{15},
			want: 16,
		},
		{
			name: "2",
			args: args{2},
			want: 2,
		},
		{
			name: "3",
			args: args{3},
			want: 4,
		},
		{
			name: "16",
			args: args{16},
			want: 16,
		},
		{
			name: "4",
			args: args{4},
			want: 4,
		},
		{
			name: "3500x3500",
			args: args{3500 * 3500 / 2},
			want: 8388608,
		},
		{
			name: "35x35",
			args: args{35 * 35},
			want: 2048,
		},
		{
			name: "490000",
			args: args{490000 / 16},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UpperPowerOfTwo(tt.args.number); got != tt.want {
				t.Errorf("UpperPowerOfTwo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClosestToPowerOfTwo(t *testing.T) {
	type args struct {
		x int32
		y int32
		n int32
	}
	tests := []struct {
		name  string
		args  args
		want  int32
		want1 int32
	}{
		{
			name: "4x7",
			args: args{
				x: 4,
				y: 7,
				n: 8,
			},
			want:  1,
			want1: 7,
		},
		{
			name:  "",
			args:  args{},
			want:  0,
			want1: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ClosestToPowerOfTwo(tt.args.x, tt.args.y, tt.args.n)
			if got != tt.want {
				t.Errorf("ClosestToPowerOfTwo() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ClosestToPowerOfTwo() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
