package errs

import (
	"errors"
	"fmt"
)

// AppError 是带业务码和 HTTP 状态的可携带错误。
type AppError struct {
	code       int
	httpStatus int
	message    string
	detail     string
	cause      error
}

// New 创建一个业务错误。
func New(code, httpStatus int, message string) *AppError {
	return &AppError{code: code, httpStatus: httpStatus, message: message}
}

// Error 实现 error 接口。
func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	if e.detail != "" {
		return fmt.Sprintf("%s: %s", e.message, e.detail)
	}
	return e.message
}

// Unwrap 支持 errors.Is / errors.As。
func (e *AppError) Unwrap() error { return e.cause }

// Code 返回业务码。
func (e *AppError) Code() int { return e.code }

// HTTPStatus 返回 HTTP 状态码。
func (e *AppError) HTTPStatus() int { return e.httpStatus }

// Message 返回用户可见的错误消息。
func (e *AppError) Message() string { return e.message }

// Detail 返回附加细节。
func (e *AppError) Detail() string { return e.detail }

// WithDetail 返回附带细节的副本。
func (e *AppError) WithDetail(detail string) *AppError {
	cp := *e
	cp.detail = detail
	return &cp
}

// WithMessage 返回覆盖消息的副本。
func (e *AppError) WithMessage(msg string) *AppError {
	cp := *e
	cp.message = msg
	return &cp
}

// Wrap 返回附带底层错误的副本。
func (e *AppError) Wrap(err error) *AppError {
	if err == nil {
		return e
	}
	cp := *e
	cp.cause = err
	return &cp
}

// From 把任意错误转换为 AppError，未知错误视为内部错误。
func From(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return ErrInternal.Wrap(err)
}

// Is 判断目标错误链中是否包含 code 相同的业务错误。
func Is(err error, target error) bool {
	return errors.Is(err, target)
}
