package utils

import "testing"

func TestParsePaginationDefaults(t *testing.T) {
	limit, offset := ParsePagination("", "")
	if limit != 50 || offset != 0 {
		t.Fatalf("got (%d, %d), want (50, 0)", limit, offset)
	}
}

func TestParsePaginationValid(t *testing.T) {
	limit, offset := ParsePagination("25", "100")
	if limit != 25 || offset != 100 {
		t.Fatalf("got (%d, %d), want (25, 100)", limit, offset)
	}
}

func TestParsePaginationClamps(t *testing.T) {
	cases := []struct{ inLimit, inOffset string; wantLimit, wantOffset int }{
		{"0", "0", 50, 0},          // zero limit falls back to default
		{"-3", "0", 50, 0},         // negative limit falls back to default
		{"101", "0", 50, 0},        // limit above cap falls back to default
		{"100", "0", 100, 0},       // limit at cap is allowed
		{"abc", "def", 50, 0},      // non-numeric falls back to defaults
		{"10", "-5", 10, 0},        // negative offset falls back to default
		{"", "0", 50, 0},           // empty offset default
	}
	for _, c := range cases {
		l, o := ParsePagination(c.inLimit, c.inOffset)
		if l != c.wantLimit || o != c.wantOffset {
			t.Errorf("ParsePagination(%q, %q) = (%d, %d), want (%d, %d)",
				c.inLimit, c.inOffset, l, o, c.wantLimit, c.wantOffset)
		}
	}
}
