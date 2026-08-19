package main

import "testing"

func almostEqual(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}

func TestWilsonIntervalZeroTotal(t *testing.T) {
	low, high := wilsonInterval(0, 0, 1.96)
	if low != 0 || high != 0 {
		t.Fatalf("wilsonInterval(0,0) = (%v, %v), want (0, 0)", low, high)
	}
}

func TestWilsonIntervalMatchesKnownExample(t *testing.T) {
	// Reference values from docs/04-api-guide.md's worked example:
	// 55/91 conversions -> conversion_rate 0.6044, CI [0.5017, 0.6987].
	low, high := wilsonInterval(55, 91, 1.96)
	if !almostEqual(low, 0.5017, 0.001) {
		t.Errorf("wilsonInterval low = %.4f, want ~0.5017", low)
	}
	if !almostEqual(high, 0.6987, 0.001) {
		t.Errorf("wilsonInterval high = %.4f, want ~0.6987", high)
	}
	if low >= high {
		t.Errorf("wilsonInterval low (%.4f) should be < high (%.4f)", low, high)
	}
}

func TestWilsonIntervalBoundsAreClamped(t *testing.T) {
	// All successes: interval must stay within [0, 1] even though the naive
	// normal-approximation formula would push the upper bound past 1.
	low, high := wilsonInterval(10, 10, 1.96)
	if low < 0 || high > 1 {
		t.Fatalf("wilsonInterval(10,10) = (%v, %v), want both within [0, 1]", low, high)
	}
	// No successes: lower bound must stay at 0, not go negative.
	low, high = wilsonInterval(0, 10, 1.96)
	if low < 0 || high > 1 {
		t.Fatalf("wilsonInterval(0,10) = (%v, %v), want both within [0, 1]", low, high)
	}
}

func TestTwoProportionZTestNoSampleReturnsNeutralPValue(t *testing.T) {
	if p := twoProportionZTest(0, 0, 5, 10); p != 1 {
		t.Errorf("twoProportionZTest with n1=0 = %v, want 1", p)
	}
	if p := twoProportionZTest(5, 10, 0, 0); p != 1 {
		t.Errorf("twoProportionZTest with n2=0 = %v, want 1", p)
	}
}

func TestTwoProportionZTestIdenticalRatesGivePValueOne(t *testing.T) {
	// Identical proportions (and n1==n2) => z==0 => two-sided p-value == 1.
	p := twoProportionZTest(30, 100, 30, 100)
	if !almostEqual(p, 1.0, 1e-9) {
		t.Errorf("twoProportionZTest with identical rates = %.6f, want 1.0", p)
	}
}

func TestTwoProportionZTestMatchesKnownExample(t *testing.T) {
	// Reference value from docs/04-api-guide.md's worked example:
	// treatment 69/109 vs control 55/91 -> p_value_vs_control 0.6778.
	p := twoProportionZTest(69, 109, 55, 91)
	if !almostEqual(p, 0.6778, 0.001) {
		t.Errorf("twoProportionZTest = %.4f, want ~0.6778", p)
	}
}

func TestTwoProportionZTestLargeDifferenceIsSignificant(t *testing.T) {
	// A stark, large-sample gap (10% vs 90% conversion) must be flagged as
	// significant at the conventional p < 0.05 threshold.
	p := twoProportionZTest(900, 1000, 100, 1000)
	if p >= 0.05 {
		t.Errorf("twoProportionZTest for a stark difference = %.6f, want < 0.05", p)
	}
}

func TestChiSquarePValueNonPositiveDfIsNeutral(t *testing.T) {
	if p := chiSquarePValue(5, 0); p != 1 {
		t.Errorf("chiSquarePValue with df=0 = %v, want 1", p)
	}
	if p := chiSquarePValue(5, -1); p != 1 {
		t.Errorf("chiSquarePValue with df=-1 = %v, want 1", p)
	}
}

func TestChiSquarePValueZeroStatisticIsNeutral(t *testing.T) {
	if p := chiSquarePValue(0, 1); p != 1 {
		t.Errorf("chiSquarePValue with chiSq=0 = %v, want 1", p)
	}
}

func TestChiSquarePValueMatchesKnownExample(t *testing.T) {
	// Reference value from docs/04-api-guide.md SRM example:
	// chi_square 1.62 with df=1 (two variants) -> p_value 0.2031, not flagged.
	p := chiSquarePValue(1.62, 1)
	if !almostEqual(p, 0.2031, 0.001) {
		t.Errorf("chiSquarePValue(1.62, 1) = %.4f, want ~0.2031", p)
	}
	if p < 0.01 {
		t.Errorf("chiSquarePValue(1.62, 1) = %.4f, should NOT be flagged (p < 0.01)", p)
	}
}

func TestChiSquarePValueLargeStatisticIsFlagged(t *testing.T) {
	// A large chi-square statistic relative to its degrees of freedom must
	// cross the SRM flagging threshold used in getExperimentStats (p < 0.01).
	p := chiSquarePValue(50, 1)
	if p >= 0.01 {
		t.Errorf("chiSquarePValue(50, 1) = %.6f, want < 0.01 (should be flagged as SRM)", p)
	}
}

func TestNormalCDFKnownValues(t *testing.T) {
	if !almostEqual(normalCDF(0), 0.5, 1e-9) {
		t.Errorf("normalCDF(0) = %v, want 0.5", normalCDF(0))
	}
	if !almostEqual(normalCDF(1.96), 0.975, 0.001) {
		t.Errorf("normalCDF(1.96) = %v, want ~0.975", normalCDF(1.96))
	}
}

func TestTwoSidedPValueSymmetric(t *testing.T) {
	if p1, p2 := twoSidedPValue(1.5), twoSidedPValue(-1.5); !almostEqual(p1, p2, 1e-9) {
		t.Errorf("twoSidedPValue(1.5)=%v, twoSidedPValue(-1.5)=%v, want equal", p1, p2)
	}
}
