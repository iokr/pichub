package dto

// UploadResp 是上传成功响应（PicGo 兼容）。
type UploadResp struct {
	ID       uint64 `json:"id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
	Markdown string `json:"markdown"`
	HTML     string `json:"html"`
	BBCode   string `json:"bbcode"`
	Size     int64  `json:"size"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Hash     string `json:"hash"`
	Dedup    bool   `json:"dedup"`
}

// UploadCheckReq 是秒传检测请求。
type UploadCheckReq struct {
	Hash string `json:"hash" binding:"required"`
	Size int64  `json:"size"`
	Name string `json:"name"`
	Ext  string `json:"ext"`
}

// UploadCheckResp 是秒传检测响应。
type UploadCheckResp struct {
	Exists bool        `json:"exists"`
	Image  *UploadResp `json:"image,omitempty"`
}

// UploadURLReq 是远程拉取请求。
type UploadURLReq struct {
	URL       string  `json:"url" binding:"required"`
	AlbumID   *uint64 `json:"album_id"`
	Tags      string  `json:"tags"`
	IsPublic  *bool   `json:"is_public"`
	StorageID uint64  `json:"storage_id"`
}

// ChunkInitReq 是分片上传初始化请求。
type ChunkInitReq struct {
	Name      string  `json:"name" binding:"required"`
	Size      int64   `json:"size" binding:"required"`
	Hash      string  `json:"hash" binding:"required"`
	Chunks    int     `json:"chunks" binding:"required"`
	AlbumID   *uint64 `json:"album_id"`
	Tags      string  `json:"tags"`
	IsPublic  *bool   `json:"is_public"`
	StorageID uint64  `json:"storage_id"`
}

// ChunkInitResp 是分片上传初始化响应。
type ChunkInitResp struct {
	UploadID  string `json:"upload_id"`
	Uploaded  []int  `json:"uploaded"`
	ChunkSize int64  `json:"chunk_size"`
}

// ChunkCompleteReq 是合并分片请求。
type ChunkCompleteReq struct {
	Hash string `json:"hash"`
}
