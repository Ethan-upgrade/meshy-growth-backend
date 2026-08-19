package experiment

import "testing"

func TestBucketRangeAndDeterminism(t *testing.T) {
	for _, userID := range []string{"user-1", "user-2", "user-3", "another-user"} {
		b1 := Bucket("exp-key", userID)
		b2 := Bucket("exp-key", userID)
		if b1 != b2 {
			t.Fatalf("Bucket(%q) not deterministic: got %d then %d", userID, b1, b2)
		}
		if b1 >= 10000 {
			t.Fatalf("Bucket(%q) = %d, want value in [0, 10000)", userID, b1)
		}
	}
}

func TestBucketHoldoutSaltIsIndependent(t *testing.T) {
	// The holdout hash uses a different salted key ("exp:holdout") than the
	// variant bucketing hash ("exp"), so they must not always coincide -
	// otherwise changing traffic_allocation would silently reshuffle
	// existing variant assignments (see docs/02-architecture.md "灰度与分桶解耦").
	differs := false
	for i := 0; i < 50; i++ {
		userID := string(rune('a' + i%26))
		bucketHash := Bucket("checkout_button_color", userID)
		holdoutHash := Bucket("checkout_button_color:holdout", userID)
		if bucketHash != holdoutHash {
			differs = true
			break
		}
	}
	if !differs {
		t.Fatal("expected bucket hash and holdout hash to diverge for at least one user")
	}
}

func TestPickVariantDeterministic(t *testing.T) {
	variants := []Variant{{Key: "control", Weight: 1}, {Key: "treatment", Weight: 1}}
	for _, userID := range []string{"u1", "u2", "u3"} {
		v1 := PickVariant("exp", userID, variants)
		v2 := PickVariant("exp", userID, variants)
		if v1 != v2 {
			t.Fatalf("PickVariant(%q) not deterministic: got %q then %q", userID, v1, v2)
		}
		if v1 != "control" && v1 != "treatment" {
			t.Fatalf("PickVariant(%q) = %q, want control or treatment", userID, v1)
		}
	}
}

func TestPickVariantEmptyOrZeroWeight(t *testing.T) {
	if got := PickVariant("exp", "u1", nil); got != "" {
		t.Fatalf("PickVariant with no variants = %q, want empty string", got)
	}
	zeroWeight := []Variant{{Key: "a", Weight: 0}, {Key: "b", Weight: 0}}
	if got := PickVariant("exp", "u1", zeroWeight); got != "" {
		t.Fatalf("PickVariant with zero total weight = %q, want empty string", got)
	}
}

func TestPickVariantRespectsWeightDistribution(t *testing.T) {
	// Statistical check over a large synthetic population: with weights 1:3,
	// "treatment" should get roughly 3x the traffic of "control" (+/-5pp).
	variants := []Variant{{Key: "control", Weight: 1}, {Key: "treatment", Weight: 3}}
	counts := map[string]int{}
	const n = 20000
	for i := 0; i < n; i++ {
		userID := randUserID(i)
		counts[PickVariant("weighted-exp", userID, variants)]++
	}
	treatmentRatio := float64(counts["treatment"]) / float64(n)
	const want = 0.75
	const tolerance = 0.02
	if diff := treatmentRatio - want; diff < -tolerance || diff > tolerance {
		t.Fatalf("treatment ratio = %.4f, want within %.2f of %.2f (counts=%v)", treatmentRatio, tolerance, want, counts)
	}
}

func randUserID(i int) string {
	// Simple deterministic pseudo-unique id generator, avoids importing
	// fmt/strconv just for test data.
	digits := []byte{'0' + byte(i/1000%10), '0' + byte(i/100%10), '0' + byte(i/10%10), '0' + byte(i%10)}
	return "user-" + string(digits)
}
