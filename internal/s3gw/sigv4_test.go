package s3gw

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/gin-gonic/gin"
)

// TestSignRequestAWSExample 使用 AWS 官方文档中的 GET Object 测试向量校验签名。
func TestSignRequestAWSExample(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "https://examplebucket.s3.amazonaws.com/test.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = "examplebucket.s3.amazonaws.com"
	r.Header.Set("Range", "bytes=0-9")
	r.Header.Set("X-Amz-Content-Sha256", emptyPayloadSHA)
	r.Header.Set("X-Amz-Date", "20130524T000000Z")

	sr := &signedRequest{
		AccessKey:     "AKIAIOSFODNN7EXAMPLE",
		Date:          "20130524",
		Region:        "us-east-1",
		Scope:         "20130524/us-east-1/s3/aws4_request",
		SignedHeaders: []string{"host", "range", "x-amz-content-sha256", "x-amz-date"},
		AmzDate:       "20130524T000000Z",
	}
	got := signRequest(r, sr, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", emptyPayloadSHA)
	want := "f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41"
	if got != want {
		t.Fatalf("signature mismatch:\n got %s\nwant %s", got, want)
	}
}

// TestSDKSignedRequest 校验网关与 AWS SDK 官方签名器的计算结果一致。
func TestSDKSignedRequest(t *testing.T) {
	body := []byte("hello pichub")
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])

	r, err := http.NewRequest(http.MethodPut, "http://127.0.0.1:8080/s3/pichub/2026/10/a.png?x=1&y=2", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "image/png")
	r.Header.Set("X-Amz-Content-Sha256", payloadHash)

	creds := aws.Credentials{AccessKeyID: "PKITESTKEY000000000", SecretAccessKey: "test-secret"}
	if err := v4.NewSigner().SignHTTP(context.Background(), creds, r, payloadHash, "s3", "us-east-1", time.Now()); err != nil {
		t.Fatal(err)
	}

	sr, aerr := parseSignedRequest(r)
	if aerr != nil {
		t.Fatalf("parse authorization: %v (raw=%q)", aerr, r.Header.Get("Authorization"))
	}
	got := signRequest(r, sr, creds.SecretAccessKey, payloadHash)
	if got != sr.Signature {
		t.Fatalf("signature mismatch with aws sdk:\n got %s\nwant %s", got, sr.Signature)
	}
}

// TestCanonicalQuery 校验查询串排序与编码。
func TestCanonicalQuery(t *testing.T) {
	q := map[string][]string{
		"list-type": {"2"},
		"prefix":    {"a b"},
		"uploads":   {""},
	}
	got := canonicalQuery(q)
	want := "list-type=2&prefix=a%20b&uploads="
	if got != want {
		t.Fatalf("canonical query = %q, want %q", got, want)
	}
}

// TestValidBucket 校验 Bucket 名称规则。
func TestValidBucket(t *testing.T) {
	cases := map[string]bool{
		"pichub":       true,
		"my-bucket.v1": true,
		"abc":          true,
		"ab":           false,
		"-abc":         false,
		"abc-":         false,
		"Abc":          false,
		"abc_":         false,
	}
	for name, want := range cases {
		if got := validBucket(name); got != want {
			t.Errorf("validBucket(%q) = %v, want %v", name, got, want)
		}
	}
}

// TestRegisterRoutes 校验网关路由注册不会冲突。
func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	g := &Gateway{prefix: "/s3"}
	g.Register(engine)
	if len(engine.Routes()) == 0 {
		t.Fatal("no routes registered")
	}
}
