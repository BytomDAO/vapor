package decimal

import (
	"github.com/pingcap/types"
	"github.com/sirupsen/logrus"
)

type Decimal struct {
	value *types.MyDecimal
}

func NewFromString(value string) (*Decimal, error) {
	dec := new(types.MyDecimal)
	if err := dec.FromString([]byte(value)); err != nil && err != types.ErrTruncated {
		return nil, err
	}

	return &Decimal{value: dec}, nil
}

func New(value int64, exp int) *Decimal {
	dec := types.NewDecFromInt(value)
	if exp < 0 {
		pow := types.NewDecFromInt(1)
		if err := pow.Shift(-exp); err != nil {
			logFatal(err,"decimal shift error")
		}

		if err := types.DecimalDiv(dec, pow, dec, 10); err != nil {
			logFatal(err,"decimal divide error")
		}
	} else {
		if err := dec.Shift(exp); err != nil {
			logFatal(err,"decimal shift error")
		}
	}
	return &Decimal{value: dec}
}

func (d *Decimal) Abs() *Decimal {
	coefficient := New(1, 0)
	if d.value.IsNegative() {
		coefficient = New(-1, 0)
	}
	return d.Mul(coefficient)
}

func (d *Decimal) Add(d2 *Decimal) *Decimal {
	result := new(types.MyDecimal)
	if err := types.DecimalAdd(d.value, d2.value, result); err != nil {
		logFatal(err,"decimal add error")
	}

	return &Decimal{value: result}
}

func (d *Decimal) Sub(d2 *Decimal) *Decimal {
	result := new(types.MyDecimal)
	if err := types.DecimalSub(d.value, d2.value, result); err != nil {
		logFatal(err,"decimal subtract error")
	}

	return &Decimal{value: result}
}

func (d *Decimal) Mul(d2 *Decimal) *Decimal {
	result := new(types.MyDecimal)
	if err := types.DecimalMul(d.value, d2.value, result); err != nil {
		logFatal(err,"decimal multiply error")
	}

	return &Decimal{value: result}
}

func (d *Decimal) Div(d2 *Decimal) *Decimal {
	result := new(types.MyDecimal)
	if err := types.DecimalDiv(d.value, d2.value, result, 10); err != nil {
		logFatal(err,"decimal divide error")
	}

	return &Decimal{value: result}
}

func (d *Decimal) Float64() float64 {
	result, err := d.value.ToFloat64()
	if err != nil {
		logFatal(err,"decimal to float64 error")
	}

	return result
}

func (d *Decimal) Int64() int64 {
	result, err := d.value.ToInt()
	if err != nil {
		logFatal(err,"decimal to int64 error")
	}

	return result
}

func (d *Decimal) Cmp(d2 *Decimal) int {
	return d.value.Compare(d2.value)
}

func (d *Decimal) LessThanOrEqual(d2 *Decimal) bool {
	return d.Cmp(d2) <= 0
}

func (d *Decimal) GreaterThan(d2 *Decimal) bool {
	return d.Cmp(d2) > 0
}

func (d *Decimal) String() string {
	return d.value.String()
}

func (d *Decimal) StringRoundFixed(places int) string {
	result := new(types.MyDecimal)
	if err := d.value.Round(result, places, types.ModeHalfEven); err != nil {
		logFatal(err, "decimal string round fixed error")
	}

	return result.String()
}

func (d *Decimal) StringCeilFixed(places int) string {
	fixedRate, err := NewFromString(d.StringRoundFixed(places))
	if err != nil {
		logFatal(err, "decimal string ceil fixed error")
	}

	if fixedRate.Cmp(d) != 0 {
		fixedRate = fixedRate.Add(New(1, -places))
	}
	return fixedRate.StringRoundFixed(places)
}

func logFatal(err error, args ...interface{}) {
	if err != types.ErrTruncated {
		logrus.WithField("err", err).Fatal(args...)
	}
}
