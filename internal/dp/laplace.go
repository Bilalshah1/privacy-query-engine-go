// Package dp implements differential privacy noise mechanisms.
package dp

import (
	"crypto/rand"
	"encoding/binary"
	"math"
)

// secureUniform returns a uniformly distributed float64 in [0, 1),
// sourced from crypto/rand rather than math/rand. DP noise generation
// should not rely on a predictable PRNG: if an attacker could predict
// or influence the noise stream, the privacy guarantee is void even
// though the math looks correct on paper.
func secureUniform() float64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failing is a fatal environment problem (no entropy
		// source) — we must not silently fall back to a weak PRNG for
		// privacy-critical noise, so panic loudly instead.
		panic("dp: crypto/rand unavailable: " + err.Error())
	}
	// Use 53 bits (a float64's mantissa) for a uniform value in [0,1).
	u64 := binary.BigEndian.Uint64(b[:])
	return float64(u64>>11) / float64(1<<53)
}

// LaplaceScale returns the scale parameter b = sensitivity / epsilon
// for the Laplace mechanism. Larger epsilon (weaker privacy) yields a
// smaller scale (less noise); smaller epsilon yields a larger scale
// (more noise).
func LaplaceScale(sensitivity, epsilon float64) float64 {
	if epsilon <= 0 {
		panic("dp: epsilon must be > 0")
	}
	return sensitivity / epsilon
}

// SampleLaplace draws a single sample from Laplace(0, scale) using the
// inverse CDF (quantile function) method:
//
//	F(x) = 0.5 * e^(x/b)         for x < 0
//	F(x) = 1 - 0.5 * e^(-x/b)    for x >= 0
//
// Inverting both branches and folding them into one expression with
// u = p - 0.5 (so u ranges over (-0.5, 0.5)) gives:
//
//	F^-1(p) = -b * sign(u) * ln(1 - 2*|u|)
//
// This is exact (no rejection, no iteration) because the Laplace CDF
// has a closed-form inverse — unlike the Gaussian, which needs
// Box-Muller instead.
func SampleLaplace(scale float64) float64 {
	u := secureUniform() - 0.5
	sign := 1.0
	if u < 0 {
		sign = -1.0
	}
	return -scale * sign * math.Log(1-2*math.Abs(u))
}

// AddLaplaceNoise adds Laplace-mechanism noise to a true numeric query
// result (typically a count) and returns the noisy value.
func AddLaplaceNoise(trueValue, sensitivity, epsilon float64) float64 {
	scale := LaplaceScale(sensitivity, epsilon)
	return trueValue + SampleLaplace(scale)
}