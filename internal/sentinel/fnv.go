package sentinel

import "fmt"

// fnv1a32 computes a 32-bit FNV-1a hash with avalanche finalizer.
// This is a custom implementation independent of hash/fnv, matching
// the Python reference implementation exactly.
func FNV1a32(text string) string {
	var h uint32 = 2166136261
	for _, c := range text {
		h ^= uint32(c)
		h *= 16777619
	}
	// Avalanche finalizer (murmur3-style)
	h ^= h >> 16
	h *= 2246822507
	h ^= h >> 13
	h *= 3266489909
	h ^= h >> 16
	return fmt.Sprintf("%08x", h)
}
