package model

import (
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func floatEq(want, got float64) bool {
	const eps = 0.0001
	return math.Abs(want-got) < eps
}

func TestAmount_ToFloat64(t *testing.T) {
	type fields struct {
		roubles int64
		kopeck  int64
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{
			"zero all",
			fields{roubles: 0, kopeck: 0},
			0.0,
		},
		{
			"zero roubles #1",
			fields{roubles: 0, kopeck: 99},
			0.99,
		},
		{
			"zero roubles #2",
			fields{roubles: 0, kopeck: 100},
			1.0,
		},
		{
			"zero roubles #3",
			fields{roubles: 0, kopeck: 1000},
			10.0,
		},
		{
			"zero roubles #4",
			fields{roubles: 0, kopeck: 123},
			1.23,
		},
		{
			"zero roubles #5",
			fields{roubles: 0, kopeck: 543},
			5.43,
		},
		{
			"zero roubles #6",
			fields{roubles: 0, kopeck: 2345},
			23.45,
		},
		{
			"many kopecks #1",
			fields{roubles: 1, kopeck: 2345},
			24.45,
		},
		{
			"many kopecks #2",
			fields{roubles: 1234, kopeck: 2345},
			1257.45,
		},
		{
			"zero kopeck #1",
			fields{roubles: 1, kopeck: 0},
			1.0,
		},
		{
			"zero kopeck #2",
			fields{roubles: 123, kopeck: 0},
			123.0,
		},
		{
			"a lot of roubles",
			fields{roubles: math.MaxInt64, kopeck: 0},
			9223372036854775807.0,
		},
		{
			"a lot of kopecks",
			fields{roubles: 0, kopeck: math.MaxInt64},
			92233720368547758.07,
		},
		{
			"a lot of all",
			fields{roubles: math.MaxInt64, kopeck: math.MaxInt64},
			9315605757223323565.07,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Amount{
				roubles: tt.fields.roubles,
				kopeck:  tt.fields.kopeck,
			}
			assert.True(t, floatEq(tt.want, a.ToFloat64()))
		})
	}
}

func TestFromFloat(t *testing.T) {
	tests := []struct {
		float   float64
		want    Amount
		wantErr bool
	}{
		{
			1234.0,
			Amount{1234, 0},
			false,
		},
		{
			0,
			Amount{0, 0},
			false,
		},
		{
			0.12345,
			Amount{0, 12},
			false,
		},
		{
			0.129999,
			Amount{0, 13},
			false,
		},
		{
			1234.164,
			Amount{1234, 16},
			false,
		},
		{
			1234.104,
			Amount{1234, 10},
			false,
		},
		{
			1234.105,
			Amount{1234, 11},
			false,
		},
		{
			1234.165,
			Amount{1234, 17},
			false,
		},
		{
			1234.175,
			Amount{1234, 18},
			false,
		},
		{
			1234.115,
			Amount{1234, 12},
			false,
		},
		{
			1234.101,
			Amount{1234, 10},
			false,
		},
		{
			1234.111,
			Amount{1234, 11},
			false,
		},
		{
			1234.167,
			Amount{1234, 17},
			false,
		},
		{
			1234.157,
			Amount{1234, 16},
			false,
		},
		{
			1234.141,
			Amount{1234, 14},
			false,
		},
		{
			1234.151,
			Amount{1234, 15},
			false,
		},
		{
			1234.145,
			Amount{1234, 15},
			false,
		},
		{
			1234.144,
			Amount{1234, 14},
			false,
		},
		{
			1234.14,
			Amount{1234, 14},
			false,
		},
		{
			1234.15,
			Amount{1234, 15},
			false,
		},
		{
			1234.990,
			Amount{1234, 99},
			false,
		},
		{
			1234.991,
			Amount{1234, 99},
			false,
		},
		{
			1234.100,
			Amount{1234, 10},
			false,
		},
		{
			1234.101,
			Amount{1234, 10},
			false,
		},
		{
			9007199254740992.01,
			Amount{0, 0},
			true,
		},
		{
			-1234.0,
			Amount{0, 0},
			true,
		},
		{
			-0.01,
			Amount{0, 0},
			true,
		},
		{
			-0.1,
			Amount{0, 0},
			true,
		},
		{
			-0.00001,
			Amount{0, 0},
			true,
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := FromFloat(tt.float)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromFloat() got = %v, want %v", got, tt.want)
			}
		})
	}
}
