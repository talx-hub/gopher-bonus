package model

import (
	"errors"
	"math"
)

type Amount struct {
	roubles int64
	kopeck  int64
}

func (a *Amount) ToFloat64() float64 {
	return float64(a.roubles) + float64(a.kopeck)/100
}

func FromFloat(amount float64) (Amount, error) {
	if amount < 0 {
		return Amount{}, errors.New("bonus amount must be positive")
	}
	const maxPreciseInt = 9007199254740992
	const kopInRub = 100
	if amount*kopInRub >= maxPreciseInt {
		return Amount{}, errors.New("amount overflow")
	}

	var a Amount
	totalKop := int64(math.Round(amount * kopInRub))
	a.roubles = totalKop / kopInRub
	a.kopeck = totalKop % kopInRub

	return a, nil
}
