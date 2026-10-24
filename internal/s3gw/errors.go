// Package s3gw 实现 S3 兼容网关：SigV4 校验、对象读写、分片上传与列举。
package s3gw

import (
	"encoding/xml"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/middleware"
)

// apiError 是 S3 协议错误。
type apiError struct {
	Code    string
	Message string
	Status  int
}

// Error 实现 error 接口。
func (e *apiError) Error() string { return e.Code + ": " + e.Message }

func newError(code, message string, status int) *apiError {
	return &apiError{Code: code, Message: message, Status: status}
}

// 常用 S3 错误。
var (
	errAccessDenied       = newError("AccessDenied", "拒绝访问", http.StatusForbidden)
	errSignatureMismatch  = newError("SignatureDoesNotMatch", "签名校验失败", http.StatusForbidden)
	errClockSkew          = newError("RequestTimeTooSkewed", "请求时间与服务器偏差过大", http.StatusForbidden)
	errReplay             = newError("RequestReplay", "请求已被处理，请勿重放", http.StatusForbidden)
	errNoSuchKey          = newError("NoSuchKey", "对象不存在", http.StatusNotFound)
	errNoSuchUpload       = newError("NoSuchUpload", "分片上传不存在或已过期", http.StatusNotFound)
	errInvalidBucket      = newError("InvalidBucketName", "Bucket 名称不合法", http.StatusBadRequest)
	errInvalidRequest     = newError("InvalidRequest", "请求参数不合法", http.StatusBadRequest)
	errMalformedXML       = newError("MalformedXML", "请求体 XML 不合法", http.StatusBadRequest)
	errEntityTooLarge     = newError("EntityTooLarge", "对象大小超出限制", http.StatusRequestEntityTooLarge)
	errContentSHA256      = newError("XAmzContentSHA256Mismatch", "请求体哈希与声明不一致", http.StatusBadRequest)
	errMethodNotAllowed   = newError("MethodNotAllowed", "不支持的方法", http.StatusMethodNotAllowed)
	errNotImplemented     = newError("NotImplemented", "暂不支持该操作", http.StatusNotImplemented)
	errInternal           = newError("InternalError", "内部错误", http.StatusInternalServerError)
	errServiceUnavailable = newError("ServiceUnavailable", "S3 网关未启用", http.StatusServiceUnavailable)
)

// s3ErrorBody 是 S3 错误响应体。
type s3ErrorBody struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	RequestID string   `xml:"RequestId"`
	HostID    string   `xml:"HostId"`
}

// fail 输出 S3 错误响应。
func (g *Gateway) fail(c *gin.Context, e *apiError) {
	if e == nil {
		e = errInternal
	}
	reqID := middleware.RequestID(c)
	if reqID == "" {
		reqID = "pichub"
	}
	c.Header("Content-Type", "application/xml")
	c.Status(e.Status)
	_, _ = c.Writer.WriteString(xml.Header)
	_ = xml.NewEncoder(c.Writer).Encode(s3ErrorBody{
		Code:      e.Code,
		Message:   e.Message,
		RequestID: reqID,
		HostID:    g.hostID,
	})
	c.Abort()
}

// mapServiceError 把业务错误映射为 S3 错误。
func mapServiceError(err error) *apiError {
	appErr := errs.From(err)
	switch appErr.Code() {
	case errs.CodeS3Signature:
		return newError("SignatureDoesNotMatch", appErr.Message(), http.StatusForbidden)
	case errs.CodeS3ClockSkew:
		return newError("RequestTimeTooSkewed", appErr.Message(), http.StatusForbidden)
	case errs.CodeS3KeyDisabled:
		return newError("AccessDenied", appErr.Message(), http.StatusForbidden)
	case errs.CodeNotFound:
		return newError("NoSuchKey", appErr.Message(), http.StatusNotFound)
	case errs.CodeForbidden:
		return newError("AccessDenied", appErr.Message(), http.StatusForbidden)
	case errs.CodeFileTooLarge:
		return newError("EntityTooLarge", appErr.Message(), http.StatusRequestEntityTooLarge)
	case errs.CodeQuotaExceeded:
		return newError("QuotaExceeded", appErr.Message(), http.StatusForbidden)
	case errs.CodeDailyLimit:
		return newError("SlowDown", appErr.Message(), http.StatusServiceUnavailable)
	case errs.CodeInvalidParam:
		msg := appErr.Message()
		if detail := appErr.Detail(); detail != "" {
			msg += ": " + detail
		}
		return newError("InvalidRequest", msg, http.StatusBadRequest)
	}
	return newError("InternalError", appErr.Message(), appErr.HTTPStatus())
}
