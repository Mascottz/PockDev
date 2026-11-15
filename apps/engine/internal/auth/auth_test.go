package auth

import (
	"testing"
	"time"
)

func TestStateRoundtrip(t *testing.T) {
	h := &Handler{ClientID: "cid", ClientSecret: "shh-secret"}
	now := time.Unix(1_700_000_000, 0)
	h.Now = func() time.Time { return now }

	redirect := "http://localhost:3000/auth/callback?x=1"
	state := h.signState(now.Unix(), "nonce123", redirect)
	got, ok := h.verifyState(state)
	if !ok {
		t.Fatalf("state %q did not verify", state)
	}
	if got != redirect {
		t.Fatalf("redirect %q want %q", got, redirect)
	}
}

func TestStateTamperRejected(t *testing.T) {
	h := &Handler{ClientID: "cid", ClientSecret: "shh-secret"}
	now := time.Unix(1_700_000_000, 0)
	h.Now = func() time.Time { return now }

	state := h.signState(now.Unix(), "nonce123", "http://localhost:3000/auth/callback")
	for _, bad := range []string{
		"",
		"garbage",
		"a.b.c.d.e",
		state[:len(state)-2] + "ff",
		"1700000000.nonce123.aGVsbG8.deadbeef",
	} {
		if _, ok := h.verifyState(bad); ok {
			t.Fatalf("state %q must not verify", bad)
		}
	}

	other := &Handler{ClientID: "cid", ClientSecret: "different"}
	other.Now = h.Now
	if _, ok := other.verifyState(state); ok {
		t.Fatal("state signed with another secret must not verify")
	}
}

func TestStateFreshness(t *testing.T) {
	h := &Handler{ClientID: "cid", ClientSecret: "shh-secret"}
	base := time.Unix(1_700_000_000, 0)
	state := h.signState(base.Unix(), "nonce123", "http://localhost:3000/auth/callback")

	h.Now = func() time.Time { return base.Add(9 * time.Minute) }
	if _, ok := h.verifyState(state); !ok {
		t.Fatal("state inside 10 minutes must verify")
	}
	h.Now = func() time.Time { return base.Add(11 * time.Minute) }
	if _, ok := h.verifyState(state); ok {
		t.Fatal("state older than 10 minutes must not verify")
	}
}

func TestTrustedRedirect(t *testing.T) {
	h := &Handler{}
	for _, ok := range []string{
		"http://localhost:3000",
		"http://localhost:3000/",
		"http://localhost:3000/auth/callback",
		"http://localhost:3000/deck?x=1",
		"https://pockdev.vercel.app/auth/callback",
		"https://pockdev.vercel.app/deck?billing=verify",
	} {
		if !h.allowedRedirect(ok) {
			t.Fatalf("%q must be trusted", ok)
		}
	}
	for _, bad := range []string{
		"http://localhost:3000.evil.com/x",
		"http://evil.com",
		"https://evil.com/http://localhost:3000",
		"http://localhost:3001/auth/callback",
		"https://pockdev.vercel.app.evil.com/x",
		"",
	} {
		if h.allowedRedirect(bad) {
			t.Fatalf("%q must not be trusted", bad)
		}
	}
}

func TestPickEmail(t *testing.T) {
	got := pickEmail([]ghEmail{
		{Email: "a@x.com", Primary: false, Verified: true},
		{Email: "p@x.com", Primary: true, Verified: true},
	})
	if got != "p@x.com" {
		t.Fatalf("primary verified wins: %q", got)
	}
	got = pickEmail([]ghEmail{
		{Email: "u@x.com", Primary: true, Verified: false},
		{Email: "v@x.com", Primary: false, Verified: true},
	})
	if got != "v@x.com" {
		t.Fatalf("verified beats unverified primary: %q", got)
	}
	if got := pickEmail(nil); got != "" {
		t.Fatalf("empty list: %q", got)
	}
}

func TestHashToken(t *testing.T) {
	if HashToken("pds_abc") != HashToken("pds_abc") {
		t.Fatal("hash must be stable")
	}
	if HashToken("pds_abc") == HashToken("pds_abd") {
		t.Fatal("hash must separate tokens")
	}
	if len(HashToken("pds_abc")) != 64 {
		t.Fatalf("sha256 hex length: %d", len(HashToken("pds_abc")))
	}
}
