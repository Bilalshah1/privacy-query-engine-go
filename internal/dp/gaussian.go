package dp

import "math"

// GaussianSigma computes the standard deviation needed for the Gaussian
// mechanism to satisfy (epsilon, delta)-differential privacy, using the
// classical analytic bound from Dwork & Roth:
//
//	sigma >= sensitivity * sqrt(2 * ln(1.25 / delta)) / epsilon
//
// This mechanism trades a small failure probability (delta) for
// tighter noise than pure Laplace on continuous aggregates like sums
// and averages, and it composes more favorably across many queries
// (noise grows roughly with sqrt(k) rather than linearly with k).
func GaussianSigma(sensitivity, epsilon, delta float64) float64 {
	if epsilon <= 0 {
		panic("dp: epsilon must be > 0")
	}
	if delta <= 0 || delta >= 1 {
		panic("dp: delta must be in (0, 1)")
	}
	return sensitivity * math.Sqrt(2*math.Log(1.25/delta)) / epsilon
}

// sampleStandardNormal draws one sample from a standard normal
// distribution N(0, 1) using the Box-Muller transform. Box-Muller is
// used here (rather than inverse CDF, as with Laplace) because the
// Gaussian CDF has no closed-form inverse — it requires the error
// function, which isn't analytically invertible.
func sampleStandardNormal() float64 {
	// Box-Muller needs u1 in (0, 1], not [0, 1), since ln(0) is undefined.
	u1 := 1 - secureUniform() // shifts [0,1) to (0,1]
	u2 := secureUniform()
	r := math.Sqrt(-2 * math.Log(u1))
	theta := 2 * math.Pi * u2
	return r * math.Cos(theta)
}

// SampleGaussian draws a single sample from N(0, sigma^2).
func SampleGaussian(sigma float64) float64 {
	return sigma * sampleStandardNormal()
}

// AddGaussianNoise adds Gaussian-mechanism noise, calibrated to
// (epsilon, delta), to a true numeric query result (typically a sum
// or average) and returns the noisy value.
func AddGaussianNoise(trueValue, sensitivity, epsilon, delta float64) float64 {
	sigma := GaussianSigma(sensitivity, epsilon, delta)
	return trueValue + SampleGaussian(sigma)
}