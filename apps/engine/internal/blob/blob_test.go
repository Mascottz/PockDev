package blob

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalPutGet(t *testing.T) {
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Put(context.Background(), "deliveries/p1/d1.body", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	got, err := l.Get(context.Background(), "deliveries/p1/d1.body")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}
	if _, err := os.Stat(filepath.Join(l.Base(), "deliveries", "p1", "d1.body")); err != nil {
		t.Fatal(err)
	}
}

func TestCleanKeyRejectsTraversal(t *testing.T) {
	for _, bad := range []string{"", ".", "..", "../x", "/../etc/passwd"} {
		if _, err := CleanKey(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
	if got, err := CleanKey("/a/b"); err != nil || got != "a/b" {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestSpoolMirrorsToRemote(t *testing.T) {
	local, _ := NewLocal(t.TempDir())
	remote, _ := NewLocal(t.TempDir())
	s := NewSpool(local, remote)

	if err := s.Put(context.Background(), "k1", []byte("data")); err != nil {
		t.Fatal(err)
	}
	// Local read works immediately.
	if got, err := s.Get(context.Background(), "k1"); err != nil || string(got) != "data" {
		t.Fatalf("local get %q %v", got, err)
	}
	// The mirror upload happens in the background.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if got, err := remote.Get(context.Background(), "k1"); err == nil && string(got) == "data" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("remote mirror never arrived")
}

func TestSpoolFallsBackToRemote(t *testing.T) {
	local, _ := NewLocal(t.TempDir())
	remote, _ := NewLocal(t.TempDir())
	s := NewSpool(local, remote)
	if err := remote.Put(context.Background(), "k2", []byte("far")); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(context.Background(), "k2")
	if err != nil || string(got) != "far" {
		t.Fatalf("got %q %v", got, err)
	}
	// It rehydrates the local copy.
	got, err = local.Get(context.Background(), "k2")
	if err != nil || string(got) != "far" {
		t.Fatalf("rehydrate %q %v", got, err)
	}
}

func TestS3SignsRequests(t *testing.T) {
	var auth, amzDate, contentHash string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		amzDate = r.Header.Get("X-Amz-Date")
		contentHash = r.Header.Get("X-Amz-Content-Sha256")
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Write([]byte("body"))
	}))
	defer srv.Close()

	s := NewS3(srv.URL, "bucket", "AK", "SK", "us-east-1")
	s.now = func() time.Time { return time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC) }

	if err := s.Put(context.Background(), "a/b.body", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(auth, "Credential=AK/20300102/us-east-1/s3/aws4_request") {
		t.Fatalf("auth %q", auth)
	}
	if !strings.Contains(auth, "SignedHeaders=content-type;host;x-amz-content-sha256;x-amz-date") {
		t.Fatalf("signed headers %q", auth)
	}
	if amzDate != "20300102T030405Z" {
		t.Fatalf("date %q", amzDate)
	}
	if contentHash == "" {
		t.Fatal("payload hash missing")
	}

	got, err := s.Get(context.Background(), "a/b.body")
	if err != nil || string(got) != "body" {
		t.Fatalf("get %q %v", got, err)
	}
}
