// Package numberjs preserves ECMAScript numeric rounding used by the reference algorithms.
package numberjs

import (
	"math"
	"math/big"
)

// Fixed2 matches Number(value.toFixed(2)), including binary fractions below a decimal tie.
func Fixed2(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return value
	}
	rational := new(big.Rat).SetFloat64(math.Abs(value))
	rational.Mul(rational, big.NewRat(100, 1))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(rational.Num(), rational.Denom(), remainder)
	if remainder.Lsh(remainder, 1).Cmp(rational.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	rounded, _ := new(big.Rat).SetFrac(quotient, big.NewInt(100)).Float64()
	return math.Copysign(rounded, value)
}
