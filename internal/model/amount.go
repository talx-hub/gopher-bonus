package model

import (
	"errors"
	"math"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
)

const kopInRub = 100

type Amount struct {
	roubles int64
	kopeck  int64
}

func NewAmount(roubles, kopeck int64) Amount {
	return Amount{
		roubles: roubles,
		kopeck:  kopeck,
	}
}

func (a *Amount) ToFloat64() float64 {
	return float64(a.roubles) + float64(a.kopeck)/kopInRub
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

	totalKop := int64(math.Round(amount * kopInRub))
	return Amount{
		roubles: totalKop / kopInRub,
		kopeck:  totalKop % kopInRub,
	}, nil
}

func (a *Amount) ToPGNumeric() pgtype.Numeric {
	return pgtype.Numeric{
		Int:   big.NewInt(a.roubles*kopInRub + a.kopeck),
		Exp:   -2,
		Valid: true,
	}
}

func FromPGNumeric(n pgtype.Numeric) (Amount, error) {
	if !n.Valid || n.NaN {
		return Amount{},
			errors.New("invalid numeric value")
	}

	totalKop := n.Int.Int64()
	return Amount{
		roubles: totalKop / kopInRub,
		kopeck:  totalKop % kopInRub,
	}, nil
}
