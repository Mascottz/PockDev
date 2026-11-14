package blob

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestSigV4OfficialS3Vector pins the signing code against the worked example
// in the AWS S3 REST authentication docs ("Example: GET Object").
func TestSigV4OfficialS3Vector(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://examplebucket.s3.amazonaws.com/test.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=0-9")
	when := time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC)

	signV4(req, nil, "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"us-east-1", "s3", when)

	want := "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request, " +
		"SignedHeaders=host;range;x-amz-content-sha256;x-amz-date, " +
		"Signature=f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41"
	got := req.Header.Get("Authorization")
	if got != want {
		t.Fatalf("authorization mismatch\n got: %s\nwant: %s", got, want)
	}
	if req.Header.Get("X-Amz-Date") != "20130524T000000Z" {
		t.Fatalf("date %q", req.Header.Get("X-Amz-Date"))
	}
	if req.Header.Get("X-Amz-Content-Sha256") != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("payload hash %q", req.Header.Get("X-Amz-Content-Sha256"))
	}
}

func TestCanonicalQuerySorted(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://x.test/p?b=2&a=1&a=0", nil)
	q := canonicalQuery(req.URL)
	if q != "a=0&a=1&b=2" && q != "a=1&a=0&b=2" {
		t.Fatalf("query %q", q)
	}
	if !strings.Contains(q, "b=2") {
		t.Fatalf("query %q", q)
	}
}
