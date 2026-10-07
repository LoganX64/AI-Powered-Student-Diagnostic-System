package utils

import (
	"encoding/hex"
	"testing"
)

func TestRandomTokenHexLength(t *testing.T) {
	for _, n := range []int{1, 16, 32, 64} {
		tok, err := RandomTokenHex(n)
		if err != nil {
			t.Fatalf("RandomTokenHex(%d): %v", n, err)
		}
		if len(tok) != 2*n {
			t.Fatalf("RandomTokenHex(%d) length = %d, want %d", n, len(tok), 2*n)
		}
		if _, err := hex.DecodeString(tok); err != nil {
			t.Fatalf("RandomTokenHex(%d) not valid hex: %v", n, err)
		}
	}
}

func TestRandomTokenHexUnique(t *testing.T) {
	a, _ := RandomTokenHex(16)
	b, _ := RandomTokenHex(16)
	if a == b {
		t.Fatal("two 16-byte tokens collided")
	}
}

func TestHashSHA256HexDeterministic(t *testing.T) {
	in := "my-reset-token"
	if got, want := HashSHA256Hex(in), HashSHA256Hex(in); got != want {
		t.Fatalf("HashSHA256Hex not deterministic: %q vs %q", got, want)
	}
	if HashSHA256Hex(in) == in {
		t.Fatal("hash must differ from input")
	}
	if HashSHA256Hex(in) == HashSHA256Hex("different") {
		t.Fatal("different inputs must hash differently")
	}
	if len(HashSHA256Hex(in)) != 64 {
		t.Fatalf("sha256 hex length = %d, want 64", len(HashSHA256Hex(in)))
	}
}
