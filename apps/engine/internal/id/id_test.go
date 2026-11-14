package id

import (
	"strings"
	"testing"
)

func TestNewFormat(t *testing.T) {
	got := New("dlv_", 24)
	if !strings.HasPrefix(got, "dlv_") {
		t.Fatalf("prefix missing: %s", got)
	}
	if len(got) != len("dlv_")+24 {
		t.Fatalf("bad length: %s", got)
	}
	other := New("dlv_", 24)
	if got == other {
		t.Fatal("ids must be unique")
	}
}

func TestToken(t *testing.T) {
	got := Token(32)
	if len(got) != 32 {
		t.Fatalf("bad length: %s", got)
	}
}
