package bvconv

import "testing"

func BenchmarkEncode(b *testing.B) {
	// Realistic AIDs covering a range of magnitudes.
	aids := []int64{
		1,
		170001,
		810872,
		1000000,
		123456789,
		9876543210,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, aid := range aids {
			_ = Encode(aid)
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	// Corresponding BVids for the AIDs above.
	bvs := []string{
		"BV1xx411c7mQ",
		"BV17x411w7KC",
		"BV1Js411o76u",
		"BV1Js411Z7Cy",
		"BV1yn411L7tG",
		"BV1eN4e1s7qR",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, bv := range bvs {
			_, _ = Decode(bv)
		}
	}
}
