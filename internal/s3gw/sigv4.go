package s3gw

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
)

const (
	sigV4Algorithm  = "AWS4-HMAC-SHA256"
	emptyPayloadSHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	unsignedPayload = "UNSIGNED-PAYLOAD"
)

// signedRequest 是解析后的 SigV4 签名信息。
type signedRequest struct {
	AccessKey     string
	Scope         string
	Date          string
	Region        string
	SignedHeaders []string
	Signature     string
	PayloadHash   string
	AmzDate       string
	Timestamp     time.Time
}

// authState 保存一次请求的鉴权上下文。
type authState struct {
	sr     *signedRequest
	cred   *model.S3Credential
	user   *model.User
	secret string
	done   bool
}

// parseSignedRequest 解析 Authorization 头。
func parseSignedRequest(r *http.Request) (*signedRequest, *apiError) {
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if raw == "" {
		if r.URL.Query().Get("X-Amz-Algorithm") != "" {
			return nil, newError("NotImplemented", "暂不支持预签名 URL", http.StatusNotImplemented)
		}
		return nil, errAccessDenied
	}
	if !strings.HasPrefix(raw, sigV4Algorithm+" ") {
		return nil, newError("InvalidRequest", "仅支持 AWS4-HMAC-SHA256 签名", http.StatusBadRequest)
	}
	rest := strings.TrimSpace(strings.TrimPrefix(raw, sigV4Algorithm+" "))
	parts := map[string]string{}
	for _, item := range strings.Split(rest, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		k, v, ok := strings.Cut(item, "=")
		if ok {
			parts[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	cred := parts["Credential"]
	signature := parts["Signature"]
	signed := parts["SignedHeaders"]
	if cred == "" || signature == "" || signed == "" {
		return nil, errAccessDenied
	}
	fields := strings.Split(cred, "/")
	if len(fields) != 5 || fields[4] != "aws4_request" || fields[3] != "s3" {
		return nil, newError("InvalidRequest", "Credential 作用域不合法", http.StatusBadRequest)
	}
	amzDate := strings.TrimSpace(r.Header.Get("X-Amz-Date"))
	ts, err := time.Parse("20060102T150405Z", amzDate)
	if err != nil {
		return nil, newError("AccessDenied", "缺少有效的 X-Amz-Date", http.StatusForbidden)
	}
	sr := &signedRequest{
		AccessKey:     fields[0],
		Date:          fields[1],
		Region:        fields[2],
		Scope:         fields[1] + "/" + fields[2] + "/s3/aws4_request",
		SignedHeaders: strings.Split(strings.ToLower(signed), ";"),
		Signature:     signature,
		AmzDate:       amzDate,
		Timestamp:     ts,
	}
	if h := strings.TrimSpace(r.Header.Get("X-Amz-Content-Sha256")); h != "" {
		sr.PayloadHash = h
	}
	return sr, nil
}

// prepareAuth 解析签名并加载凭据；载荷哈希已知时立即完成校验。
func (g *Gateway) prepareAuth(c *gin.Context) (*authState, *apiError) {
	sr, aerr := parseSignedRequest(c.Request)
	if aerr != nil {
		return nil, aerr
	}
	if skew := g.cfg.S3Gateway.ClockSkew; skew > 0 {
		diff := time.Since(sr.Timestamp)
		if diff < -skew || diff > skew {
			return nil, errClockSkew
		}
	}
	cred, secret, user, err := g.svc.Token.ValidateS3Credential(c.Request.Context(), sr.AccessKey)
	if err != nil {
		return nil, mapServiceError(err)
	}
	st := &authState{sr: sr, cred: cred, user: user, secret: secret}
	if strings.HasPrefix(sr.PayloadHash, "STREAMING-") {
		return nil, newError("NotImplemented", "暂不支持流式签名载荷", http.StatusNotImplemented)
	}
	if sr.PayloadHash != "" && sr.PayloadHash != unsignedPayload {
		if aerr := g.finishAuth(c, st, sr.PayloadHash); aerr != nil {
			return nil, aerr
		}
	}
	return st, nil
}

// authenticate 完成无请求体的鉴权（GET/HEAD/DELETE）。
func (g *Gateway) authenticate(c *gin.Context) (*authState, *apiError) {
	st, aerr := g.prepareAuth(c)
	if aerr != nil {
		return nil, aerr
	}
	if !st.done {
		if aerr := g.finishAuth(c, st, emptyPayloadSHA); aerr != nil {
			return nil, aerr
		}
	}
	return st, nil
}

// finishAuth 用实际载荷哈希完成签名与防重放校验。
func (g *Gateway) finishAuth(c *gin.Context, st *authState, payloadHash string) *apiError {
	if st.done {
		return nil
	}
	if payloadHash == "" {
		payloadHash = emptyPayloadSHA
	}
	// 客户端声明 UNSIGNED-PAYLOAD 时，签名中的载荷哈希就是该字面量。
	if st.sr.PayloadHash == unsignedPayload {
		payloadHash = unsignedPayload
	}
	expected := signRequest(c.Request, st.sr, st.secret, payloadHash)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(st.sr.Signature)) != 1 {
		return errSignatureMismatch
	}
	// 只对会改变状态的请求做防重放，避免同名 GET/HEAD 在同一秒内被误判。
	if g.replay != nil && g.cfg.S3Gateway.ReplayWindow > 0 &&
		c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		key := "s3sig:" + hashString(st.sr.AccessKey+"|"+st.sr.Signature)
		first, rerr := g.replay.MarkOnce(c.Request.Context(), key, g.cfg.S3Gateway.ReplayWindow)
		if rerr != nil {
			if g.log != nil {
				g.log.Warn("s3 replay check failed", logger.Err(rerr))
			}
		} else if !first {
			return errReplay
		}
	}
	st.done = true
	g.svc.Token.TouchS3Credential(c.Request.Context(), st.cred.ID, c.ClientIP())
	return nil
}

// verifyPayload 校验请求体哈希与声明一致。
func (g *Gateway) verifyPayload(st *authState, actual string) *apiError {
	declared := st.sr.PayloadHash
	if declared == "" || declared == unsignedPayload {
		return nil
	}
	if strings.HasPrefix(declared, "STREAMING-") {
		return errNotImplemented
	}
	if !strings.EqualFold(declared, actual) {
		return errContentSHA256
	}
	return nil
}

// signRequest 计算 SigV4 签名。
func signRequest(r *http.Request, sr *signedRequest, secret, payloadHash string) string {
	canonical := strings.Join([]string{
		r.Method,
		canonicalURI(r),
		canonicalQuery(r.URL.Query()),
		canonicalHeaders(r, sr.SignedHeaders),
		strings.Join(sr.SignedHeaders, ";"),
		payloadHash,
	}, "\n")
	stringToSign := strings.Join([]string{
		sigV4Algorithm,
		sr.AmzDate,
		sr.Scope,
		hashString(canonical),
	}, "\n")
	key := hmacSHA256([]byte("AWS4"+secret), sr.Date)
	key = hmacSHA256(key, sr.Region)
	key = hmacSHA256(key, "s3")
	key = hmacSHA256(key, "aws4_request")
	return hex.EncodeToString(hmacSHA256(key, stringToSign))
}

func canonicalURI(r *http.Request) string {
	if p := r.URL.EscapedPath(); p != "" {
		return p
	}
	return "/"
}

func canonicalQuery(q url.Values) string {
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		values := append([]string(nil), q[k]...)
		sort.Strings(values)
		for _, v := range values {
			if sb.Len() > 0 {
				sb.WriteByte('&')
			}
			sb.WriteString(awsEncode(k))
			sb.WriteByte('=')
			sb.WriteString(awsEncode(v))
		}
	}
	return sb.String()
}

func canonicalHeaders(r *http.Request, signed []string) string {
	var sb strings.Builder
	for _, name := range signed {
		var value string
		switch name {
		case "host":
			value = r.Host
		case "content-length":
			if r.ContentLength >= 0 {
				value = strconv.FormatInt(r.ContentLength, 10)
			}
		default:
			values := r.Header.Values(name)
			if len(values) == 0 {
				values = r.Header.Values(http.CanonicalHeaderKey(name))
			}
			value = strings.Join(values, ",")
		}
		value = strings.Join(strings.Fields(value), " ")
		sb.WriteString(name)
		sb.WriteByte(':')
		sb.WriteString(value)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// awsEncode 按 RFC3986 对签名元素编码。
func awsEncode(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '.' || ch == '_' || ch == '~' {
			sb.WriteByte(ch)
			continue
		}
		sb.WriteByte('%')
		sb.WriteString(strings.ToUpper(hex.EncodeToString([]byte{ch})))
	}
	return sb.String()
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
