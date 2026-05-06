package bvconv

import (
	"math/rand"
	"testing"
	"time"
)

func TestKnownPairs(t *testing.T) {
	pairs := []struct {
		aid int64
		bv  string
	}{
		{1, "BV1xx411c7mQ"},
		{2, "BV1xx411c7mD"},
		{3, "BV1xx411c7mS"},
		{170001, "BV17x411w7KC"},
		{810872, "BV1Js411o76u"},
		{1000000, "BV1Js411Z7Cy"},
		{123456789, "BV1yn411L7tG"},
		{9876543210, "BV1eN4e1s7qR"},
	}

	for _, p := range pairs {
		t.Run("encode_"+p.bv, func(t *testing.T) {
			got := Encode(p.aid)
			if got != p.bv {
				t.Errorf("Encode(%d) = %s, want %s", p.aid, got, p.bv)
			}
		})

		t.Run("decode_"+p.bv, func(t *testing.T) {
			got, err := Decode(p.bv)
			if err != nil {
				t.Fatalf("Decode(%s) error: %v", p.bv, err)
			}
			if got != p.aid {
				t.Errorf("Decode(%s) = %d, want %d", p.bv, got, p.aid)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 1000; i++ {
		aid := rng.Int63n(maxAid-minAid) + minAid
		bv := Encode(aid)
		decoded, err := Decode(bv)
		if err != nil {
			t.Fatalf("Decode(Encode(%d)) error: %v", aid, err)
		}
		if decoded != aid {
			t.Errorf("Decode(Encode(%d)) = %d", aid, decoded)
		}
	}
}

func TestDecodeVariants(t *testing.T) {
	bv := "BV1xx411c7mQ"
	variants := []string{
		bv,
		"bv1xx411c7mQ",
		"Bv1xx411c7mQ",
		"  BV1xx411c7mQ  ",
		"xx411c7mQ",
	}

	for _, v := range variants {
		got, err := Decode(v)
		if err != nil {
			t.Fatalf("Decode(%q) error: %v", v, err)
		}
		if got != 1 {
			t.Errorf("Decode(%q) = %d, want 1", v, got)
		}
	}
}

func TestEncodeBoundary(t *testing.T) {
	t.Run("min_aid", func(t *testing.T) {
		got := Encode(1)
		if got == "" {
			t.Error("Encode(1) returned empty string")
		}
	})

	t.Run("max_aid_minus_1", func(t *testing.T) {
		got := Encode(maxAid - 1)
		if got == "" {
			t.Error("Encode(maxAid-1) returned empty string")
		}
	})
}

func TestEncodePanics(t *testing.T) {
	cases := []int64{0, -1, -100, maxAid, maxAid + 1, maxAid + 100}
	for _, aid := range cases {
		t.Run("panic", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Encode(%d) should have panicked", aid)
				}
			}()
			Encode(aid)
		})
	}
}

func TestDecodeErrors(t *testing.T) {
	cases := []string{
		"",
		"BV1",
		"BV1short",
		"BV1xx411c7mQextra",
		"BV1xx411c7m!",
		"invalid!!!",
	}

	for _, c := range cases {
		t.Run("error_"+c, func(t *testing.T) {
			_, err := Decode(c)
			if err == nil {
				t.Errorf("Decode(%q) should return error", c)
			}
		})
	}
}
