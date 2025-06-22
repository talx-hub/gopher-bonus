package model

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAmount(t *testing.T) {
	type fields struct {
		roubles int64
		kopecks int64
	}
	tests := []struct {
		name   string
		fields fields
		want   Amount
	}{
		{
			"zero all",
			fields{roubles: 0, kopecks: 0},
			Amount{
				roubles: 0,
				kopeck:  0,
			},
		},
		{
			"zero roubles #1",
			fields{roubles: 0, kopecks: 99},
			Amount{
				roubles: 0,
				kopeck:  99,
			},
		},
		{
			"zero roubles #2",
			fields{roubles: 0, kopecks: 100},
			Amount{
				roubles: 1,
				kopeck:  0,
			},
		},
		{
			"zero roubles #3",
			fields{roubles: 0, kopecks: 1000},
			Amount{
				roubles: 10,
				kopeck:  0,
			},
		},
		{
			"zero roubles #4",
			fields{roubles: 0, kopecks: 123},
			Amount{
				roubles: 1,
				kopeck:  23,
			},
		},
		{
			"zero roubles #5",
			fields{roubles: 0, kopecks: 543},
			Amount{
				roubles: 5,
				kopeck:  43,
			},
		},
		{
			"zero roubles #6",
			fields{roubles: 0, kopecks: 2345},
			Amount{
				roubles: 23,
				kopeck:  45,
			},
		},
		{
			"zero roubles #7",
			fields{roubles: 0, kopecks: 10},
			Amount{
				roubles: 0,
				kopeck:  10,
			},
		},
		{
			"many kopecks #1",
			fields{roubles: 1, kopecks: 2345},
			Amount{
				roubles: 24,
				kopeck:  45,
			},
		},
		{
			"many kopecks #2",
			fields{roubles: 1234, kopecks: 2345},
			Amount{
				roubles: 1257,
				kopeck:  45,
			},
		},
		{
			"zero kopeck #1",
			fields{roubles: 1, kopecks: 0},
			Amount{
				roubles: 1,
				kopeck:  0,
			},
		},
		{
			"zero kopeck #2",
			fields{roubles: 123, kopecks: 0},
			Amount{
				roubles: 123,
				kopeck:  0,
			},
		},
		{
			"a lot of roubles",
			fields{roubles: math.MaxInt64, kopecks: 0},
			Amount{
				roubles: math.MaxInt64,
				kopeck:  0,
			},
		},
		{
			"a lot of kopecks",
			fields{roubles: 0, kopecks: math.MaxInt64},
			Amount{
				roubles: 92233720368547758,
				kopeck:  7,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAmount(tt.fields.roubles, tt.fields.kopecks)
			assert.Equal(t, tt.want, a)
		})
	}
}

func TestAmount_String(t *testing.T) {
	tests := []struct {
		name     string
		input    Amount
		expected string
	}{
		{
			name:     "without kopecks",
			input:    Amount{roubles: 7, kopeck: 0},
			expected: "7",
		},
		{
			name:     "with one-digit kopecks",
			input:    Amount{roubles: 10, kopeck: 5},
			expected: "10.05",
		},
		{
			name:     "with two-digit kopecks",
			input:    Amount{roubles: 1, kopeck: 42},
			expected: "1.42",
		},
		{
			name:     "zero amount",
			input:    Amount{roubles: 0, kopeck: 0},
			expected: "0",
		},
		{
			name:     "kopecks only",
			input:    Amount{roubles: 0, kopeck: 99},
			expected: "0.99",
		},
		{
			name:     "round number with kopecks",
			input:    Amount{roubles: 3, kopeck: 10},
			expected: "3.10",
		},
		{
			name:     "large number",
			input:    Amount{roubles: 123456789, kopeck: 1},
			expected: "123456789.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input.String())
		})
	}
}

func TestFromString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Amount
		expectError bool
	}{
		{
			name:     "whole roubles only",
			input:    "123",
			expected: Amount{roubles: 123, kopeck: 0},
		},
		{
			name:     "roubles and kopecks",
			input:    "123.45",
			expected: Amount{roubles: 123, kopeck: 45},
		},
		{
			name:     "one-digit kopecks",
			input:    "7.5",
			expected: Amount{roubles: 7, kopeck: 50},
		},
		{
			name:     "two-digit kopecks #2",
			input:    "7.05",
			expected: Amount{roubles: 7, kopeck: 5},
		},
		{
			name:        "too many decimal places",
			input:       "10.123",
			expectError: true,
		},
		{
			name:        "invalid roubles part",
			input:       "abc.10",
			expectError: true,
		},
		{
			name:        "invalid kopecks part",
			input:       "10.ab",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "multiple dots",
			input:       "10.50.20",
			expectError: true,
		},
		{
			name:        "empty roubles",
			input:       ".12",
			expectError: true,
		},
		{
			name:        "empty kopecks",
			input:       "12.",
			expectError: true,
		},
		{
			name:     "zero amount #1",
			input:    "0",
			expected: Amount{roubles: 0, kopeck: 0},
		},
		{
			name:     "zero amount #2",
			input:    "0.0",
			expected: Amount{roubles: 0, kopeck: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := FromString(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, ErrFromString))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, amount)
			}
		})
	}
}
