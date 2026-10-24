// Package errs 定义统一的业务错误码与错误类型。
package errs

import "net/http"

// 业务错误码。
const (
	CodeOK              = 0
	CodeInvalidParam    = 40001
	CodeUsernameTaken   = 40010
	CodeEmailTaken      = 40011
	CodeUsernameBad     = 40012
	CodePasswordWeak    = 40013
	CodeCaptchaWrong    = 40014
	CodeInviteInvalid   = 40015
	CodeCaptchaNeeded   = 40020
	CodeUnauthorized    = 40100
	CodeTokenExpired    = 40101
	CodeForbidden       = 40300
	CodeEmailUnverified = 40301
	CodeEmailRequired   = 40302
	CodeS3Signature     = 40310
	CodeS3ClockSkew     = 40311
	CodeS3KeyDisabled   = 40312
	CodeNotFound        = 40400
	CodeFileTooLarge    = 41300
	CodeQuotaExceeded   = 41301
	CodeDailyLimit      = 41302
	CodeTooManyReqs     = 42900
	CodeInternal        = 50000
)

// 预定义错误。
var (
	ErrInvalidParam    = New(CodeInvalidParam, http.StatusBadRequest, "参数错误")
	ErrUsernameTaken   = New(CodeUsernameTaken, http.StatusConflict, "用户名已占用")
	ErrEmailTaken      = New(CodeEmailTaken, http.StatusConflict, "邮箱已注册")
	ErrUsernameInvalid = New(CodeUsernameBad, http.StatusBadRequest, "用户名不合法")
	ErrPasswordWeak    = New(CodePasswordWeak, http.StatusBadRequest, "密码强度不足")
	ErrCaptchaWrong    = New(CodeCaptchaWrong, http.StatusBadRequest, "验证码错误或已过期")
	ErrInviteInvalid   = New(CodeInviteInvalid, http.StatusBadRequest, "邀请码无效")
	ErrCaptchaRequired = New(CodeCaptchaNeeded, http.StatusBadRequest, "需要验证码")
	ErrUnauthorized    = New(CodeUnauthorized, http.StatusUnauthorized, "未登录")
	ErrTokenExpired    = New(CodeTokenExpired, http.StatusUnauthorized, "Token 过期")
	ErrForbidden       = New(CodeForbidden, http.StatusForbidden, "无权限")
	ErrEmailUnverified = New(CodeEmailUnverified, http.StatusForbidden, "邮箱未验证")
	ErrEmailRequired   = New(CodeEmailRequired, http.StatusForbidden, "请先完成邮箱验证")
	ErrS3Signature     = New(CodeS3Signature, http.StatusForbidden, "S3 签名无效")
	ErrS3ClockSkew     = New(CodeS3ClockSkew, http.StatusForbidden, "S3 时间偏差过大")
	ErrS3KeyDisabled   = New(CodeS3KeyDisabled, http.StatusForbidden, "S3 密钥已禁用")
	ErrNotFound        = New(CodeNotFound, http.StatusNotFound, "资源不存在")
	ErrFileTooLarge    = New(CodeFileTooLarge, http.StatusRequestEntityTooLarge, "文件超限")
	ErrQuotaExceeded   = New(CodeQuotaExceeded, http.StatusRequestEntityTooLarge, "容量配额超限")
	ErrDailyLimit      = New(CodeDailyLimit, http.StatusRequestEntityTooLarge, "每日上传数超限")
	ErrTooManyRequests = New(CodeTooManyReqs, http.StatusTooManyRequests, "请求过于频繁")
	ErrInternal        = New(CodeInternal, http.StatusInternalServerError, "内部错误")
)
