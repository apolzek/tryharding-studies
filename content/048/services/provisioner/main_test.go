package main

import "testing"

func TestRandHexLength(t *testing.T) {
	for _, n := range []int{8, 16, 32} {
		got := randHex(n)
		if len(got) != n {
			t.Errorf("randHex(%d) len=%d", n, len(got))
		}
	}
}

func TestRandHexDistinct(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		v := randHex(24)
		if seen[v] {
			t.Fatalf("collision %s at i=%d", v, i)
		}
		seen[v] = true
	}
}
