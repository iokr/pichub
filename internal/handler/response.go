package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
)

// OK 返回成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, dto.Response{Code: errs.CodeOK, Message: "ok", Data: data})
}

// NoContent 返回空成功响应。
func NoContent(c *gin.Context, data any) {
	OK(c, data)
}

// Fail 返回统一错误响应。
func Fail(c *gin.Context, err error) {
	appErr := errs.From(err)
	if appErr == nil {
		appErr = errs.ErrInternal
	}
	if err != nil {
		// 原始错误只进日志，响应体仅暴露安全消息。
		_ = c.Error(err)
	}
	c.AbortWithStatusJSON(appErr.HTTPStatus(), dto.Response{
		Code:    appErr.Code(),
		Message: appErr.Message(),
	})
}

// FailMessage 以指定错误码返回自定义消息。
func FailMessage(c *gin.Context, appErr *errs.AppError, message string) {
	if appErr == nil {
		appErr = errs.ErrInternal
	}
	c.AbortWithStatusJSON(appErr.HTTPStatus(), dto.Response{
		Code:    appErr.Code(),
		Message: message,
	})
}

// BindJSON 绑定 JSON 请求体，失败时写出参数错误响应。
func BindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		Fail(c, errs.ErrInvalidParam.WithDetail(err.Error()))
		return false
	}
	return true
}

// BindQuery 绑定查询参数，失败时写出参数错误响应。
func BindQuery(c *gin.Context, obj any) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		Fail(c, errs.ErrInvalidParam.WithDetail(err.Error()))
		return false
	}
	return true
}

// pathID 解析路径参数中的无符号整数 ID。
func pathID(c *gin.Context, name string) (uint64, bool) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || v == 0 {
		Fail(c, errs.ErrInvalidParam.WithDetail("ID 不合法"))
		return 0, false
	}
	return v, true
}

// queryUint 解析查询参数中的无符号整数。
func queryUint(c *gin.Context, name string) uint64 {
	v, _ := strconv.ParseUint(c.Query(name), 10, 64)
	return v
}

// queryInt 解析查询参数中的整数。
func queryInt(c *gin.Context, name string, def int) int {
	raw := c.Query(name)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

// queryInt8 解析可空的有符号 8 位整数。
func queryInt8(c *gin.Context, name string) *int8 {
	raw := c.Query(name)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 8)
	if err != nil {
		return nil
	}
	out := int8(v)
	return &out
}
