package types

// Copyright 2015 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"math"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"
)

// RoundFloat rounds float val to the nearest integer value with float64 format, like MySQL Round function.
// RoundFloat uses default rounding mode, see https://dev.mysql.com/doc/refman/5.7/en/precision-math-rounding.html
// so rounding use "round half away from zero".
// e.g, 1.5 -> 2, -1.5 -> -2.
func RoundFloat(f float64) float64 {
	if math.Abs(f) < 0.5 {
		return 0
	}

	return math.Trunc(f + math.Copysign(0.5, f))
}

// Round rounds the argument f to dec decimal places.
// dec defaults to 0 if not specified. dec can be negative
// to cause dec digits left of the decimal point of the
// value f to become zero.
func Round(f float64, dec int) float64 {
	shift := math.Pow10(dec)
	tmp := f * shift
	if math.IsInf(tmp, 0) {
		return f
	}
	return RoundFloat(tmp) / shift
}

// Truncate truncates the argument f to dec decimal places.
// dec defaults to 0 if not specified. dec can be negative
// to cause dec digits left of the decimal point of the
// value f to become zero.
func Truncate(f float64, dec int) float64 {
	shift := math.Pow10(dec)
	tmp := f * shift
	if math.IsInf(tmp, 0) {
		return f
	}
	return math.Trunc(tmp) / shift
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func myMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func myMaxInt8(a, b int8) int8 {
	if a > b {
		return a
	}
	return b
}

func myMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func myMinInt8(a, b int8) int8 {
	if a < b {
		return a
	}
	return b
}

const (
	maxUint    = uint64(math.MaxUint64)
	uintCutOff = maxUint/uint64(10) + 1
	intCutOff  = uint64(math.MaxInt64) + 1
)

// strToInt converts a string to an integer in best effort.
func strToInt(str string) (int64, error) {
	str = strings.TrimSpace(str)
	if len(str) == 0 {
		return 0, ErrTruncated
	}
	negative := false
	i := 0
	if str[i] == '-' {
		negative = true
		i++
	} else if str[i] == '+' {
		i++
	}

	var (
		err    error
		hasNum = false
	)
	r := uint64(0)
	for ; i < len(str); i++ {
		if !unicode.IsDigit(rune(str[i])) {
			err = ErrTruncated
			break
		}
		hasNum = true
		if r >= uintCutOff {
			r = 0
			err = ErrBadNumber
			break
		}
		r = r * uint64(10)

		r1 := r + uint64(str[i]-'0')
		if r1 < r || r1 > maxUint {
			r = 0
			err = ErrBadNumber
			break
		}
		r = r1
	}
	if !hasNum {
		err = ErrTruncated
	}

	if !negative && r >= intCutOff {
		return math.MaxInt64, ErrBadNumber
	}

	if negative && r > intCutOff {
		return math.MinInt64, ErrBadNumber
	}

	if negative {
		r = -r
	}
	return int64(r), err
}

// Log logs the error if it is not nil.
func Log(err error) {
	if err != nil {
		logrus.Error("encountered error", err)
	}
}
