package s3gw

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
	"github.com/pichub/pichub/internal/service"
)

// defaultBucket 是网关对外暴露的逻辑 Bucket 名称。
const defaultBucket = "pichub"

// ReplayGuard 是签名防重放抽象。
type ReplayGuard interface {
	MarkOnce(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

// Gateway 是 S3 兼容网关。
type Gateway struct {
	cfg    *config.Config
	log    *logger.Logger
	svc    *service.Services
	replay ReplayGuard
	parts  *multipartStore
	prefix string
	hostID string
}

// New 创建 S3 网关。
func New(cfg *config.Config, log *logger.Logger, svc *service.Services, replay ReplayGuard) *Gateway {
	prefix := strings.TrimRight(strings.TrimSpace(cfg.S3Gateway.PathPrefix), "/")
	if prefix == "" {
		prefix = "/s3"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	g := &Gateway{
		cfg:    cfg,
		log:    log,
		svc:    svc,
		replay: replay,
		parts:  newMultipartStore(),
		prefix: prefix,
	}
	if svc != nil && svc.Settings != nil {
		g.hostID = svc.Settings.SiteName(context.Background())
	}
	return g
}

// Register 注册网关路由。
func (g *Gateway) Register(r *gin.Engine) {
	r.Any(g.prefix, g.handle)
	r.Any(g.prefix+"/*path", g.handle)
}

// handle 是网关统一入口，按路径与方法分发。
func (g *Gateway) handle(c *gin.Context) {
	ctx := c.Request.Context()
	if !g.svc.Settings.GetBool(ctx, service.KeyS3GatewayEnabled, g.cfg.S3Gateway.Enabled) {
		g.fail(c, errServiceUnavailable)
		return
	}
	bucket, key := g.parsePath(c)
	if bucket == "" {
		if c.Request.Method == http.MethodGet {
			g.listBuckets(c)
			return
		}
		g.fail(c, errMethodNotAllowed)
		return
	}
	if !validBucket(bucket) {
		g.fail(c, errInvalidBucket)
		return
	}
	if key == "" {
		g.handleBucket(c, bucket)
		return
	}
	g.handleObject(c, bucket, key)
}

// parsePath 从原始请求路径解析 Bucket 与对象 Key。
func (g *Gateway) parsePath(c *gin.Context) (bucket, key string) {
	p := c.Request.URL.EscapedPath()
	if !strings.HasPrefix(p, g.prefix) {
		return "", ""
	}
	rest := strings.TrimLeft(strings.TrimPrefix(p, g.prefix), "/")
	if rest == "" {
		return "", ""
	}
	parts := strings.SplitN(rest, "/", 2)
	bucket, _ = url.PathUnescape(parts[0])
	if len(parts) == 2 {
		key, _ = url.PathUnescape(parts[1])
	}
	return bucket, key
}

func (g *Gateway) handleBucket(c *gin.Context, bucket string) {
	q := c.Request.URL.Query()
	switch {
	case c.Request.Method == http.MethodHead:
		if _, aerr := g.authenticate(c); aerr != nil {
			g.fail(c, aerr)
			return
		}
		c.Header("x-amz-bucket-region", g.region())
		c.Status(http.StatusOK)
	case c.Request.Method == http.MethodGet && q.Has("location"):
		if _, aerr := g.authenticate(c); aerr != nil {
			g.fail(c, aerr)
			return
		}
		g.writeXML(c, http.StatusOK, locationConstraint{
			Xmlns:              s3Namespace,
			LocationConstraint: g.region(),
		})
	case c.Request.Method == http.MethodGet:
		g.listObjects(c, bucket)
	default:
		g.fail(c, errMethodNotAllowed)
	}
}

func (g *Gateway) handleObject(c *gin.Context, bucket, key string) {
	q := c.Request.URL.Query()
	method := c.Request.Method
	uploadID := q.Get("uploadId")
	partNumber := q.Get("partNumber")
	switch {
	case method == http.MethodPut && uploadID != "" && partNumber != "":
		g.uploadPart(c, bucket, key, uploadID, partNumber)
	case method == http.MethodPut:
		g.putObject(c, bucket, key)
	case method == http.MethodPost && q.Has("uploads"):
		g.createMultipart(c, bucket, key)
	case method == http.MethodPost && uploadID != "":
		g.completeMultipart(c, bucket, key, uploadID)
	case method == http.MethodDelete && uploadID != "":
		g.abortMultipart(c, bucket, key, uploadID)
	case method == http.MethodGet:
		g.getObject(c, key, false)
	case method == http.MethodHead:
		g.getObject(c, key, true)
	case method == http.MethodDelete:
		g.deleteObject(c, key)
	default:
		g.fail(c, errMethodNotAllowed)
	}
}

func (g *Gateway) putObject(c *gin.Context, bucket, key string) {
	st, aerr := g.prepareAuth(c)
	if aerr != nil {
		g.fail(c, aerr)
		return
	}
	tmp, size, sha, md5hex, aerr := g.savePayload(c, g.maxObjectSize(st.user))
	if aerr != nil {
		g.fail(c, aerr)
		return
	}
	defer os.Remove(tmp)
	if aerr := g.verifyPayload(st, sha); aerr != nil {
		g.fail(c, aerr)
		return
	}
	if aerr := g.finishAuth(c, st, sha); aerr != nil {
		g.fail(c, aerr)
		return
	}
	f, err := os.Open(tmp)
	if err != nil {
		g.fail(c, errInternal)
		return
	}
	defer f.Close()
	if _, uerr := g.svc.Upload.UploadObject(c.Request.Context(), st.user, key, f, size, c.ClientIP(), c.Request.UserAgent()); uerr != nil {
		g.fail(c, mapServiceError(uerr))
		return
	}
	c.Header("ETag", `"`+md5hex+`"`)
	c.Status(http.StatusOK)
}

func (g *Gateway) getObject(c *gin.Context, key string, head bool) {
	ctx := c.Request.Context()
	var (
		obj *service.S3Object
		err error
	)
	if st, aerr := g.authenticate(c); aerr == nil {
		obj, err = g.svc.Image.OpenS3Object(ctx, st.user.ID, key)
	} else if aerr == errAccessDenied && c.GetHeader("Authorization") == "" && !c.Request.URL.Query().Has("X-Amz-Signature") {
		// 匿名下载仅允许公开图片，便于 PicGo customDomain 直接访问。
		obj, err = g.svc.Image.OpenPublicS3Object(ctx, key)
	} else {
		g.fail(c, aerr)
		return
	}
	if err != nil {
		g.fail(c, mapServiceError(err))
		return
	}
	img := obj.Image
	c.Header("ETag", `"`+img.Hash+`"`)
	c.Header("Last-Modified", img.CreatedAt.UTC().Format(http.TimeFormat))
	c.Header("Content-Type", img.Mime)
	c.Header("Accept-Ranges", "bytes")
	c.Header("x-amz-storage-class", "STANDARD")
	if head {
		c.Header("Content-Length", strconv.FormatInt(img.Size, 10))
		c.Status(http.StatusOK)
		return
	}
	if obj.LocalPath != "" {
		f, oerr := os.Open(obj.LocalPath)
		if oerr != nil {
			g.fail(c, errNoSuchKey)
			return
		}
		defer f.Close()
		http.ServeContent(c.Writer, c.Request, filepath.Base(obj.LocalPath), img.CreatedAt, f)
		return
	}
	if obj.Reader == nil {
		g.fail(c, errNoSuchKey)
		return
	}
	defer obj.Reader.Close()
	if img.Size > 0 {
		c.Header("Content-Length", strconv.FormatInt(img.Size, 10))
	}
	c.Status(http.StatusOK)
	if _, cerr := io.Copy(c.Writer, obj.Reader); cerr != nil && g.log != nil {
		g.log.Warn("s3 get object failed", logger.Err(cerr), logger.String("key", key))
	}
}

func (g *Gateway) deleteObject(c *gin.Context, key string) {
	st, aerr := g.authenticate(c)
	if aerr != nil {
		g.fail(c, aerr)
		return
	}
	if err := g.svc.Image.DeleteS3Object(c.Request.Context(), st.user.ID, key); err != nil {
		// 与 AWS 语义一致：对象不存在也返回 204。
		if mapServiceError(err).Code == errNoSuchKey.Code {
			c.Status(http.StatusNoContent)
			return
		}
		g.fail(c, mapServiceError(err))
		return
	}
	c.Status(http.StatusNoContent)
}

func (g *Gateway) listBuckets(c *gin.Context) {
	if _, aerr := g.authenticate(c); aerr != nil {
		g.fail(c, aerr)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	g.writeXML(c, http.StatusOK, listBucketsResult{
		Xmlns:   s3Namespace,
		Owner:   bucketOwner{ID: "pichub", DisplayName: g.hostID},
		Buckets: []bucketEntry{{Name: defaultBucket, CreationDate: now}},
	})
}

func (g *Gateway) listObjects(c *gin.Context, bucket string) {
	st, aerr := g.authenticate(c)
	if aerr != nil {
		g.fail(c, aerr)
		return
	}
	q := c.Request.URL.Query()
	prefix := q.Get("prefix")
	delimiter := q.Get("delimiter")
	maxKeys := 1000
	if raw := q.Get("max-keys"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			maxKeys = n
			if maxKeys > 1000 {
				maxKeys = 1000
			}
		}
	}
	after := ""
	if token := q.Get("continuation-token"); token != "" {
		if raw, derr := base64.RawURLEncoding.DecodeString(token); derr == nil {
			after = string(raw)
		}
	}
	result := listBucketResult{
		Xmlns:   s3Namespace,
		Name:    bucket,
		Prefix:  prefix,
		MaxKeys: maxKeys,
	}
	if maxKeys > 0 {
		// 带分隔符时多取一些，尽量填满一页。
		fetch := maxKeys
		if delimiter != "" {
			fetch = 1000
		}
		list, truncated, err := g.svc.Image.ListS3Objects(c.Request.Context(), st.user.ID, prefix, after, fetch)
		if err != nil {
			g.fail(c, mapServiceError(err))
			return
		}
		lastKey := ""
		seen := map[string]bool{}
		for i := 0; i < len(list); i++ {
			k := list[i].StorageKey
			cp := commonPrefixOf(k, prefix, delimiter)
			if cp != "" {
				if seen[cp] {
					lastKey = k
					continue
				}
				if result.KeyCount >= maxKeys {
					truncated = true
					break
				}
				seen[cp] = true
				result.CommonPrefixes = append(result.CommonPrefixes, commonPrefix{Prefix: cp})
				result.KeyCount++
				lastKey = k
				for i+1 < len(list) && strings.HasPrefix(list[i+1].StorageKey, cp) {
					i++
					lastKey = list[i].StorageKey
				}
				continue
			}
			if result.KeyCount >= maxKeys {
				truncated = true
				break
			}
			img := list[i]
			result.Contents = append(result.Contents, listObject{
				Key:          img.StorageKey,
				LastModified: img.CreatedAt.UTC().Format(time.RFC3339),
				ETag:         `"` + img.Hash + `"`,
				Size:         img.Size,
				StorageClass: "STANDARD",
			})
			result.KeyCount++
			lastKey = img.StorageKey
		}
		result.IsTruncated = truncated
		if truncated && lastKey != "" {
			result.NextContinuationToken = base64.RawURLEncoding.EncodeToString([]byte(lastKey))
		}
	}
	g.writeXML(c, http.StatusOK, result)
}

func (g *Gateway) createMultipart(c *gin.Context, bucket, key string) {
	if !g.cfg.S3Gateway.AllowMultipart {
		g.fail(c, errNotImplemented)
		return
	}
	st, aerr := g.authenticate(c)
	if aerr != nil {
		g.fail(c, aerr)
		return
	}
	up, err := g.parts.create(st.user.ID, bucket, key)
	if err != nil {
		g.fail(c, errInternal)
		return
	}
	g.writeXML(c, http.StatusOK, initiateMultipartUploadResult{
		Xmlns:    s3Namespace,
		Bucket:   bucket,
		Key:      key,
		UploadID: up.ID,
	})
}

func (g *Gateway) uploadPart(c *gin.Context, bucket, key, uploadID, partNumber string) {
	if !g.cfg.S3Gateway.AllowMultipart {
		g.fail(c, errNotImplemented)
		return
	}
	number, err := strconv.Atoi(partNumber)
	if err != nil || number < 1 || number > 10000 {
		g.fail(c, newError("InvalidArgument", "分片序号不合法", http.StatusBadRequest))
		return
	}
	st, aerr := g.prepareAuth(c)
	if aerr != nil {
		g.fail(c, aerr)
		return
	}
	up := g.parts.get(uploadID)
	if up == nil || up.UserID != st.user.ID || up.Bucket != bucket || up.Key != key {
		g.fail(c, errNoSuchUpload)
		return
	}
	tmp, size, sha, md5hex, aerr := g.savePayload(c, g.maxObjectSize(st.user))
	if aerr != nil {
		g.fail(c, aerr)
		return
	}
	if aerr := g.verifyPayload(st, sha); aerr != nil {
		os.Remove(tmp)
		g.fail(c, aerr)
		return
	}
	if aerr := g.finishAuth(c, st, sha); aerr != nil {
		os.Remove(tmp)
		g.fail(c, aerr)
		return
	}
	if serr := g.parts.savePart(up, number, tmp, size, md5hex); serr != nil {
		os.Remove(tmp)
		g.fail(c, errInternal)
		return
	}
	c.Header("ETag", `"`+md5hex+`"`)
	c.Status(http.StatusOK)
}

func (g *Gateway) completeMultipart(c *gin.Context, bucket, key, uploadID string) {
	if !g.cfg.S3Gateway.AllowMultipart {
		g.fail(c, errNotImplemented)
		return
	}
	st, aerr := g.prepareAuth(c)
	if aerr != nil {
		g.fail(c, aerr)
		return
	}
	up := g.parts.get(uploadID)
	if up == nil || up.UserID != st.user.ID || up.Bucket != bucket || up.Key != key {
		g.fail(c, errNoSuchUpload)
		return
	}
	body, rerr := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if rerr != nil {
		g.fail(c, newError("IncompleteBody", "读取请求体失败", http.StatusBadRequest))
		return
	}
	sum := hashBytes(body)
	if aerr := g.verifyPayload(st, sum); aerr != nil {
		g.fail(c, aerr)
		return
	}
	if aerr := g.finishAuth(c, st, sum); aerr != nil {
		g.fail(c, aerr)
		return
	}
	var req completeMultipartUpload
	if len(strings.TrimSpace(string(body))) > 0 {
		if uerr := xml.Unmarshal(body, &req); uerr != nil {
			g.fail(c, errMalformedXML)
			return
		}
	}
	tmp, total, etag, aerr2 := g.parts.assemble(up, req.Parts)
	if aerr2 != nil {
		g.fail(c, newError("InvalidPart", aerr2.Error(), http.StatusBadRequest))
		return
	}
	defer os.Remove(tmp)
	f, oerr := os.Open(tmp)
	if oerr != nil {
		g.fail(c, errInternal)
		return
	}
	defer f.Close()
	if _, uerr := g.svc.Upload.UploadObject(c.Request.Context(), st.user, key, f, total, c.ClientIP(), c.Request.UserAgent()); uerr != nil {
		g.fail(c, mapServiceError(uerr))
		return
	}
	g.parts.drop(uploadID)
	g.writeXML(c, http.StatusOK, completeMultipartUploadResult{
		Xmlns:    s3Namespace,
		Location: g.objectURL(bucket, key),
		Bucket:   bucket,
		Key:      key,
		ETag:     `"` + etag + `"`,
	})
}

func (g *Gateway) abortMultipart(c *gin.Context, bucket, key, uploadID string) {
	st, aerr := g.authenticate(c)
	if aerr != nil {
		g.fail(c, aerr)
		return
	}
	up := g.parts.get(uploadID)
	if up == nil || up.UserID != st.user.ID || up.Bucket != bucket || up.Key != key {
		g.fail(c, errNoSuchUpload)
		return
	}
	g.parts.drop(uploadID)
	c.Status(http.StatusNoContent)
}

// savePayload 把请求体写入临时文件并计算 SHA-256 与 MD5。
func (g *Gateway) savePayload(c *gin.Context, limit int64) (string, int64, string, string, *apiError) {
	f, err := os.CreateTemp("", "pichub-s3-*.tmp")
	if err != nil {
		return "", 0, "", "", errInternal
	}
	name := f.Name()
	sha := sha256.New()
	md5sum := md5.New()
	var reader io.Reader = c.Request.Body
	if limit > 0 {
		reader = io.LimitReader(reader, limit+1)
	}
	n, cerr := io.Copy(io.MultiWriter(f, sha, md5sum), reader)
	closeErr := f.Close()
	if cerr != nil {
		os.Remove(name)
		return "", 0, "", "", newError("IncompleteBody", "读取请求体失败", http.StatusBadRequest)
	}
	if closeErr != nil {
		os.Remove(name)
		return "", 0, "", "", errInternal
	}
	if limit > 0 && n > limit {
		os.Remove(name)
		return "", 0, "", "", errEntityTooLarge
	}
	return name, n, hex.EncodeToString(sha.Sum(nil)), hex.EncodeToString(md5sum.Sum(nil)), nil
}

func (g *Gateway) maxObjectSize(user *model.User) int64 {
	limit := g.cfg.Upload.MaxFileSize
	if limit <= 0 {
		limit = 10 << 20
	}
	if user != nil && user.MaxFileSize > 0 {
		limit = user.MaxFileSize
	}
	return limit
}

func (g *Gateway) region() string {
	if r := strings.TrimSpace(g.cfg.S3Gateway.Region); r != "" {
		return r
	}
	return "us-east-1"
}

func (g *Gateway) objectURL(bucket, key string) string {
	return g.cfg.SiteURL() + g.prefix + "/" + bucket + "/" + strings.TrimLeft(key, "/")
}

func (g *Gateway) writeXML(c *gin.Context, status int, v any) {
	c.Header("Content-Type", "application/xml")
	c.Status(status)
	_, _ = c.Writer.WriteString(xml.Header)
	_ = xml.NewEncoder(c.Writer).Encode(v)
}

// validBucket 校验 Bucket 名称。
func validBucket(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '.' {
			continue
		}
		return false
	}
	first, last := name[0], name[len(name)-1]
	firstOK := (first >= 'a' && first <= 'z') || (first >= '0' && first <= '9')
	lastOK := (last >= 'a' && last <= 'z') || (last >= '0' && last <= '9')
	return firstOK && lastOK
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// commonPrefixOf 返回 Key 相对 Prefix 的首个分隔符片段，无分隔符时返回空。
func commonPrefixOf(key, prefix, delimiter string) string {
	if delimiter == "" || !strings.HasPrefix(key, prefix) {
		return ""
	}
	rest := key[len(prefix):]
	i := strings.Index(rest, delimiter)
	if i < 0 {
		return ""
	}
	return prefix + rest[:i+len(delimiter)]
}

// s3Namespace 是 S3 XML 命名空间。
const s3Namespace = "http://s3.amazonaws.com/doc/2006-03-01/"

type locationConstraint struct {
	XMLName            xml.Name `xml:"LocationConstraint"`
	Xmlns              string   `xml:"xmlns,attr"`
	LocationConstraint string   `xml:",chardata"`
}

type bucketOwner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type bucketEntry struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type listBucketsResult struct {
	XMLName xml.Name      `xml:"ListAllMyBucketsResult"`
	Xmlns   string        `xml:"xmlns,attr"`
	Owner   bucketOwner   `xml:"Owner"`
	Buckets []bucketEntry `xml:"Buckets>Bucket"`
}

type listObject struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

type listBucketResult struct {
	XMLName               xml.Name       `xml:"ListBucketResult"`
	Xmlns                 string         `xml:"xmlns,attr"`
	Name                  string         `xml:"Name"`
	Prefix                string         `xml:"Prefix"`
	KeyCount              int            `xml:"KeyCount"`
	MaxKeys               int            `xml:"MaxKeys"`
	IsTruncated           bool           `xml:"IsTruncated"`
	Contents              []listObject   `xml:"Contents"`
	CommonPrefixes        []commonPrefix `xml:"CommonPrefixes"`
	NextContinuationToken string         `xml:"NextContinuationToken,omitempty"`
}

// commonPrefix 是带分隔符列举时的公共前缀。
type commonPrefix struct {
	Prefix string `xml:"Prefix"`
}

type initiateMultipartUploadResult struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	Xmlns    string   `xml:"xmlns,attr"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadID string   `xml:"UploadId"`
}

type completeMultipartUpload struct {
	XMLName xml.Name       `xml:"CompleteMultipartUpload"`
	Parts   []completePart `xml:"Part"`
}

type completePart struct {
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
}

type completeMultipartUploadResult struct {
	XMLName  xml.Name `xml:"CompleteMultipartUploadResult"`
	Xmlns    string   `xml:"xmlns,attr"`
	Location string   `xml:"Location"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	ETag     string   `xml:"ETag"`
}
