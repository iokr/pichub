// Package dto 定义 HTTP 请求与响应结构。
package dto

// Response 是统一响应结构。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// PageResp 是分页响应。
type PageResp struct {
	List   any    `json:"list"`
	Total  int64  `json:"total"`
	Cursor uint64 `json:"cursor,omitempty"`
	Size   int    `json:"size,omitempty"`
}

// PageQuery 是通用分页查询参数。
type PageQuery struct {
	Page int `form:"page" json:"page"`
	Size int `form:"size" json:"size"`
}

// IDReq 是路径 ID 参数。
type IDReq struct {
	ID uint64 `uri:"id" binding:"required"`
}

// IDsReq 是批量 ID 请求。
type IDsReq struct {
	IDs []uint64 `json:"ids" binding:"required"`
}
