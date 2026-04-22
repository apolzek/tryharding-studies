package main

import "testing"

func TestValidEmail(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"a@b.co", true},
		{"foo.bar@example.com", true},
		{"no-at-sign", false},
		{"@nope.com", false},
		{"trailing@", false},
		{"nodot@examplecom", false},
		{"", false},
	}
	for _, c := range cases {
		if got := validEmail(c.in); got != c.ok {
			t.Errorf("validEmail(%q) = %v, want %v", c.in, got, c.ok)
		}
	}
}

func TestRandPasswordLength(t *testing.T) {
	for _, n := range []int{16, 24, 32} {
		p := randPassword(n)
		if len(p) != n {
			t.Errorf("randPassword(%d) len=%d", n, len(p))
		}
	}
}

func TestNewULIDUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := newULID()
		if seen[id] {
			t.Fatalf("duplicate ulid at i=%d: %s", i, id)
		}
		seen[id] = true
	}
}

func TestIsUniqueViolation(t *testing.T) {
	if !isUniqueViolation(errString("ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)")) {
		t.Fatal("expected true")
	}
	if isUniqueViolation(errString("some other error")) {
		t.Fatal("expected false")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
