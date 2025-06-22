package model

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

const kopInRub = 100

type Amount struct {
	roubles int64
	kopeck  int64
}

func NewAmount(roubles, kopeck int64) Amount {
	return Amount{
		roubles: roubles + kopeck/kopInRub,
		kopeck:  kopeck % kopInRub,
	}
}

func (a *Amount) String() string {
	if a.kopeck == 0 {
		return strconv.FormatInt(a.roubles, 10)
	}
	return fmt.Sprintf("%d.%02d", a.roubles, a.kopeck)
}

var ErrFromString = errors.New("failed to parse amount from string")

func FromString(number string) (Amount, error) {
	const errFmt = "%w: %s"
	parts := strings.Split(number, ".")
	l := len(parts)
	if l == 0 || l > 2 {
		return Amount{}, fmt.Errorf(errFmt, ErrFromString, number)
	}
	const roublesIdx = 0
	rubs, err := strconv.ParseInt(parts[roublesIdx], 10, 64)
	if err != nil {
		return Amount{}, fmt.Errorf(errFmt, ErrFromString, number)
	}
	if len(parts) == 1 {
		return NewAmount(rubs, 0), nil
	}
	const kopeckIdx = 1
	const maxCorrectNumberOfDigits = 2
	if len(parts[kopeckIdx]) > maxCorrectNumberOfDigits {
		return Amount{}, fmt.Errorf("%w: %s -- incorrect precision", ErrFromString, number)
	}
	kops, err := strconv.ParseInt(parts[kopeckIdx], 10, 64)
	if err != nil {
		return Amount{}, fmt.Errorf(errFmt, ErrFromString, number)
	}

	const twoDigit = 2
	if len(parts[kopeckIdx]) == twoDigit {
		return NewAmount(rubs, kops), nil
	}

	const factor = 10 // "10.1" -> 10rub 10kop
	return NewAmount(rubs, kops*factor), nil
}

func (a *Amount) ToPGNumeric() pgtype.Numeric {
	return pgtype.Numeric{
		Int:   big.NewInt(a.roubles*kopInRub + a.kopeck),
		Exp:   -2,
		Valid: true,
	}
}

func (a *Amount) TotalKopecks() int64 {
	return a.roubles*kopInRub + a.kopeck
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
