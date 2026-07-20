package dp

import (
	"math"
	"testing"
)

func TestLaplaceUnbiased(t *testing.T) {
	const scale = 5.0
	const n = 200000
	sum := 0.0
	sumAbs := 0.0
	for i := 0; i < n; i++ {
		x := SampleLaplace(scale)
		sum += x
		sumAbs += math.Abs(x)
	}
	mean := sum / n
	meanAbs := sumAbs / n

	if math.Abs(mean) > 0.15 {
		t.Errorf("Laplace mean should be near 0, got %f", mean)
	}
	if math.Abs(meanAbs-scale) > 0.15 {
		t.Errorf("Laplace mean(|X|) should be near scale=%f, got %f", scale, meanAbs)
	}
}

func TestGaussianUnbiased(t *testing.T) {
	const sigma = 3.0
	const n = 200000
	sum := 0.0
	sumSq := 0.0
	for i := 0; i < n; i++ {
		x := SampleGaussian(sigma)
		sum += x
		sumSq += x * x
	}
	mean := sum / n
	variance := sumSq/n - mean*mean
	stddev := math.Sqrt(variance)

	if math.Abs(mean) > 0.1 {
		t.Errorf("Gaussian mean should be near 0, got %f", mean)
	}
	if math.Abs(stddev-sigma) > 0.1 {
		t.Errorf("Gaussian stddev should be near %f, got %f", sigma, stddev)
	}
}

func TestSmallEpsilonMeansMoreNoise(t *testing.T) {
	sensitivity := 1.0
	scaleStrong := LaplaceScale(sensitivity, 0.1)
	scaleWeak := LaplaceScale(sensitivity, 10.0)

	if scaleStrong <= scaleWeak {
		t.Errorf("expected scale for epsilon=0.1 (%f) > scale for epsilon=10 (%f)", scaleStrong, scaleWeak)
	}
}