package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
)

// Captcha 生成图片验证码。
func (h *Handler) Captcha(c *gin.Context) {
	resp, err := h.svc.Auth.Captcha(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// Register 注册新用户。
func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterReq
	if !BindJSON(c, &req) {
		return
	}
	ip, ua := h.meta(c)
	req.IP = ip
	resp, err := h.svc.Auth.Register(c.Request.Context(), &req, ip, ua)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// Login 登录。
func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginReq
	if !BindJSON(c, &req) {
		return
	}
	ip, ua := h.meta(c)
	req.IP = ip
	resp, err := h.svc.Auth.Login(c.Request.Context(), &req, ip, ua)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// Refresh 刷新令牌。
func (h *Handler) Refresh(c *gin.Context) {
	var req dto.RefreshReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Auth.Refresh(c.Request.Context(), &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// Logout 登出。
func (h *Handler) Logout(c *gin.Context) {
	var req dto.LogoutReq
	// 请求体可选。
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.Auth.Logout(c.Request.Context(), h.claims(c), req.RefreshToken); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// CheckUsername 检查用户名可用性。
func (h *Handler) CheckUsername(c *gin.Context) {
	name := c.Query("u")
	if name == "" {
		name = c.Query("username")
	}
	resp, err := h.svc.Auth.CheckUsername(c.Request.Context(), name)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// CheckEmail 检查邮箱可用性。
func (h *Handler) CheckEmail(c *gin.Context) {
	email := c.Query("e")
	if email == "" {
		email = c.Query("email")
	}
	resp, err := h.svc.Auth.CheckEmail(c.Request.Context(), email)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// VerifyEmail 验证邮箱。
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailReq
	if c.Request.Method == http.MethodGet || c.ContentType() == "" {
		req.Token = c.Query("token")
	} else if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	if req.Token == "" {
		Fail(c, errs.ErrInvalidParam.WithDetail("缺少验证令牌"))
		return
	}
	if err := h.svc.Auth.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// ResendVerify 重发验证邮件。
func (h *Handler) ResendVerify(c *gin.Context) {
	var req dto.ResendVerifyReq
	if !BindJSON(c, &req) {
		return
	}
	if err := h.svc.Auth.ResendVerify(c.Request.Context(), &req); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// ForgotPassword 发送重置密码邮件。
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordReq
	if !BindJSON(c, &req) {
		return
	}
	if err := h.svc.Auth.ForgotPassword(c.Request.Context(), &req); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// ResetPassword 重置密码。
func (h *Handler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordReq
	if !BindJSON(c, &req) {
		return
	}
	if err := h.svc.Auth.ResetPassword(c.Request.Context(), &req); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}
