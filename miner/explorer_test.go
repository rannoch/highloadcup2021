package miner

import "testing"

func Test_exploreCost(t *testing.T) {
	type args struct {
		areaSize int32
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "1",
			args: args{1},
			want: 1,
		},
		{
			name: "2",
			args: args{2},
			want: 1,
		},
		{
			name: "3",
			args: args{3},
			want: 1,
		},
		{
			name: "4",
			args: args{4},
			want: 2,
		},
		{
			name: "7",
			args: args{7},
			want: 2,
		},
		{
			name: "8",
			args: args{8},
			want: 3,
		},
		{
			name: "31",
			args: args{31},
			want: 4,
		},
		{
			name: "32",
			args: args{32},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exploreCost(tt.args.areaSize); got != tt.want {
				t.Errorf("exploreCost() = %v, want %v", got, tt.want)
			}
		})
	}
}
