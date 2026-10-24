package handler

import (
	"github.com/gin-gonic/gin"
)

// StatsTopReferers 返回 TOP 引用域名。
func (h *Handler) StatsTopReferers(c *gin.Context) {
	limit := queryInt(c, "limit", 10)
	days := queryInt(c, "days", 30)
	list, err := h.svc.Stats.TopReferers(c.Request.Context(), h.userID(c), limit, days)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// StatsRefererTrend 返回引用趋势。
func (h *Handler) StatsRefererTrend(c *gin.Context) {
	days := queryInt(c, "days", 30)
	list, err := h.svc.Stats.RefererTrend(c.Request.Context(), h.userID(c), days)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}
