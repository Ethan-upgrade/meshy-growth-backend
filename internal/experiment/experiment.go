package experiment

import (
	"github.com/spaolacci/murmur3"
)

// Bucket returns a stable value in [0, 10000) for a given experiment key and
// user id, used for deterministic traffic splitting.
func Bucket(experimentKey, userID string) uint32 {
	h := murmur3.Sum32([]byte(experimentKey + ":" + userID))
	return h % 10000
}

type Variant struct {
	Key    string
	Weight int
}

// PickVariant deterministically assigns a user to one of the given variants,
// proportionally to their weight.
func PickVariant(experimentKey, userID string, variants []Variant) string {
	totalWeight := 0
	for _, v := range variants {
		totalWeight += v.Weight
	}
	if totalWeight == 0 || len(variants) == 0 {
		return ""
	}
	bucket := int(Bucket(experimentKey, userID))
	cursor := 0
	for _, v := range variants {
		cursor += v.Weight
		threshold := cursor * 10000 / totalWeight
		if bucket < threshold {
			return v.Key
		}
	}
	return variants[len(variants)-1].Key
}
