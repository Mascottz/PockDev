package blob

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// S3 talks to an S3 compatible bucket (Rimbra Buckets) with SigV4 signing,
// implemented here to keep the dependency list minimal.
type S3 struct {
	endpoint  string
	bucket    string
	accessKey string
	secretKey string
	region    string
	client    *http.Client
	now       func() time.Time
}

// NewS3 builds an S3 client. endpoint is the service base URL without a
// bucket prefix, for example https://s3.rimbra.app.
func NewS3(endpoint, bucket, accessKey, secretKey, region string) *S3 {
	if region == "" {
		region = "us-east-1"
	}
	return &S3{
		endpoint:  strings.TrimSuffix(endpoint, "/"),
		bucket:    bucket,
		accessKey: accessKey,
		secretKey: secretKey,
		region:    region,
		client:    &http.Client{Timeout: 30 * time.Second},
		now:       time.Now,
	}
}

func (s *S3) objectURL(key string) (string, error) {
	clean, err := CleanKey(key)
	if err != nil {
		return "", err
	}
	return s.endpoint + "/" + s.bucket + "/" + clean, nil
}

// Put uploads the body to the bucket.
func (s *S3) Put(ctx context.Context, key string, data []byte) error {
	u, err := s.objectURL(key)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	signV4(req, data, s.accessKey, s.secretKey, s.region, "s3", s.now().UTC())
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("s3 put %s: %s %s", key, resp.Status, string(body))
	}
	return nil
}

// Get downloads the body from the bucket.
func (s *S3) Get(ctx context.Context, key string) ([]byte, error) {
	u, err := s.objectURL(key)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	signV4(req, nil, s.accessKey, s.secretKey, s.region, "s3", s.now().UTC())
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("s3 get %s: %s", key, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024))
}

// Delete removes the body from the bucket. A missing key is not an error.
func (s *S3) Delete(ctx context.Context, key string) error {
	u, err := s.objectURL(key)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	signV4(req, nil, s.accessKey, s.secretKey, s.region, "s3", s.now().UTC())
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("s3 delete %s: %s", key, resp.Status)
	}
	return nil
}

// signV4 adds SigV4 headers to req. The payload hash covers data (nil means an
// empty payload). Every header on the request is signed.
func signV4(req *http.Request, data []byte, accessKey, secretKey, region, service string, t time.Time) {
	payloadHash := sha256Hex(data)
	amzDate := t.Format("20060102T150405Z")
	dateStamp := t.Format("20060102")

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	headers := map[string]string{
		"host":                 req.URL.Host,
		"x-amz-content-sha256": payloadHash,
		"x-amz-date":           amzDate,
	}
	for k, vals := range req.Header {
		lk := strings.ToLower(k)
		switch lk {
		case "authorization", "user-agent", "accept", "accept-encoding", "connection", "content-length":
			continue
		}
		if len(vals) > 0 {
			headers[lk] = vals[0]
		}
	}

	canonicalHeaders, signedHeaders := canonicalHeaders(headers)
	canonicalRequest := strings.Join([]string{
		req.Method,
		escapedPath(req.URL),
		canonicalQuery(req.URL),
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := strings.Join([]string{dateStamp, region, service, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	key := hmacSHA256([]byte("AWS4"+secretKey), dateStamp)
	key = hmacSHA256(key, region)
	key = hmacSHA256(key, service)
	key = hmacSHA256(key, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(key, stringToSign))

	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, scope, signedHeaders, signature))
}

func canonicalHeaders(h map[string]string) (string, string) {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var cb strings.Builder
	for _, k := range keys {
		cb.WriteString(k)
		cb.WriteString(":")
		cb.WriteString(strings.TrimSpace(h[k]))
		cb.WriteString("\n")
	}
	return cb.String(), strings.Join(keys, ";")
}

func canonicalQuery(u *url.URL) string {
	q := u.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		vals := q[k]
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

func escapedPath(u *url.URL) string {
	if u.EscapedPath() == "" {
		return "/"
	}
	return u.EscapedPath()
}

func sha256Hex(data []byte) string {
	if data == nil {
		data = []byte{}
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(data))
	return m.Sum(nil)
}
