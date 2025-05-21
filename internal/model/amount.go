package model

type Amount struct {
	roubles int64
	kopeck  int64
}

func (a Amount) ToFloat64() float64 {
	return float64(a.roubles) + float64(a.kopeck)/100
}
