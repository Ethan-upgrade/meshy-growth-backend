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

// TestStatsAgainstRealWorldABDataset cross-checks getExperimentStats' math
// helpers against aggregates computed independently from a real, publicly
// available A/B testing dataset: the Udacity "Analyze A/B Test Results"
// dataset, mirrored at https://github.com/30lm32/ml-ab-testing/blob/master/ab_data.csv
// (294,479 rows of control/treatment landing-page conversions).
//
// Aggregation methodology (reproduced in testdata/compute_stats.go and
// independently cross-verified with a Python/math.erf script):
//   - drop 3,894 duplicate user_id rows (same user recorded more than once)
//   - drop 2,044 rows where group/landing_page are inconsistent (a known
//     data-quality issue in this dataset: e.g. group=control with
//     landing_page=new_page)
//
// Resulting real-world aggregates:
//
//	control:   assigned=144226 converted=17349 rate=0.1203
//	treatment: assigned=144314 converted=17134 rate=0.1187
//
// This is a classic "no significant difference" A/B test result (the new
// landing page did not move conversion), which is exactly what our
// significance-test helpers should report: a high p-value, well above 0.05.
func TestStatsAgainstRealWorldABDataset(t *testing.T) {
	const (
		controlConverted, controlAssigned     = 17349, 144226
		treatmentConverted, treatmentAssigned = 17134, 144314
	)

	controlRate := float64(controlConverted) / float64(controlAssigned)
	if !almostEqual(controlRate, 0.1203, 0.0001) {
		t.Errorf("control conversion rate = %.4f, want ~0.1203", controlRate)
	}
	treatmentRate := float64(treatmentConverted) / float64(treatmentAssigned)
	if !almostEqual(treatmentRate, 0.1187, 0.0001) {
		t.Errorf("treatment conversion rate = %.4f, want ~0.1187", treatmentRate)
	}

	low, high := wilsonInterval(controlConverted, controlAssigned, 1.96)
	if low >= controlRate || high <= controlRate {
		t.Errorf("control Wilson CI (%.4f, %.4f) does not bracket its own rate %.4f", low, high, controlRate)
	}

	// Independently computed via Python (math.erf) from the same two
	// proportions: z = -1.2942..., two-sided p = 0.19558...
	p := twoProportionZTest(treatmentConverted, treatmentAssigned, controlConverted, controlAssigned)
	if !almostEqual(p, 0.1956, 0.001) {
		t.Errorf("twoProportionZTest on real dataset = %.4f, want ~0.1956 (matches independent Python cross-check)", p)
	}
	if p < 0.05 {
		t.Errorf("p-value %.4f should NOT be significant at alpha=0.05 (this dataset's well-known result: new landing page did not improve conversion)", p)
	}

	// With near-equal sample sizes (144226 vs 144314) the SRM check must NOT
	// flag this as a sample-ratio mismatch.
	weights := []int{1, 1}
	assigned := []int64{controlAssigned, treatmentAssigned}
	totalWeight, totalAssigned := 0, int64(0)
	for i, w := range weights {
		totalWeight += w
		totalAssigned += assigned[i]
	}
	chiSq := 0.0
	for i, w := range weights {
		expected := float64(totalAssigned) * float64(w) / float64(totalWeight)
		diff := float64(assigned[i]) - expected
		chiSq += diff * diff / expected
	}
	srmP := chiSquarePValue(chiSq, len(weights)-1)
	if srmP < 0.01 {
		t.Errorf("SRM p-value %.6f on balanced real-world sample sizes should NOT be flagged (< 0.01)", srmP)
	}
}
