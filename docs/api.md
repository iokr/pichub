# PicHub 后端服务接口文档

本文档依据 `internal/router`、`internal/handler`、`internal/dto`、`internal/service` 与 `internal/s3gw`
的实际实现编写，覆盖 PicHub 对外提供的全部 HTTP 接口，包括 REST API、图片直出、健康检查与 S3 兼容网关。

## 目录

- [1. 通用约定](#1-通用约定)
  - [1.1 基础地址](#11-基础地址)
  - [1.2 统一响应结构](#12-统一响应结构)
  - [1.3 分页结构](#13-分页结构)
  - [1.4 认证方式](#14-认证方式)
  - [1.5 限流规则](#15-限流规则)
  - [1.6 业务错误码](#16-业务错误码)
  - [1.7 通用枚举](#17-通用枚举)
- [2. 公共接口](#2-公共接口)
- [3. 认证模块](#3-认证模块)
- [4. 用户模块](#4-用户模块)
- [5. 上传模块](#5-上传模块)
- [6. 图片模块](#6-图片模块)
- [7. 相册模块](#7-相册模块)
- [8. 标签模块](#8-标签模块)
- [9. API 令牌与 S3 凭据](#9-api-令牌与-s3-凭据)
- [10. 分享模块](#10-分享模块)
- [11. 存储策略模块](#11-存储策略模块)
- [12. 引用统计模块](#12-引用统计模块)
- [13. 管理端模块](#13-管理端模块)
- [14. 图片直出](#14-图片直出)
- [15. S3 兼容网关](#15-s3-兼容网关)
- [16. 附录：数据结构](#16-附录数据结构)

---

## 1. 通用约定

### 1.1 基础地址

| 项目 | 值 |
| --- | --- |
| API 前缀 | `/api/v1` |
| 站点地址 | 由配置 `site.url` 或系统设置 `site.url` 决定，例如 `https://img.example.com` |
| 传输格式 | 请求与响应均为 JSON（上传接口为 `multipart/form-data`，S3 网关为 XML） |
| 时间格式 | RFC 3339 / ISO 8601，UTC 时区，例如 `2026-10-25T08:30:00Z` |
| 请求标识 | 每个响应都带 `X-Request-ID` 响应头，可用于排查问题 |

### 1.2 统一响应结构

除图片直出、健康检查与 S3 网关外，所有接口返回统一结构：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | int | 业务错误码，`0` 表示成功 |
| `message` | string | 提示信息，成功时为 `ok` |
| `data` | any | 业务数据，无数据时该字段省略 |

失败响应示例（HTTP 状态码与业务码同时体现错误类型）：

```json
{
  "code": 40100,
  "message": "未登录"
}
```

限流失败时额外返回当前窗口计数：

```json
{
  "code": 42900,
  "message": "请求过于频繁",
  "data": { "count": 11 }
}
```

### 1.3 分页结构

分页接口的 `data` 统一为：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `list` | array | 当前页数据 |
| `total` | int64 | 总条数；游标模式下为 `0`（不统计） |
| `cursor` | uint64 | 下一页游标，仅在游标模式下返回 |
| `size` | int | 每页条数，请求未传时可能省略 |

分页参数通用约定：

| 参数 | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `page` | int | `1` | 页码，从 1 开始 |
| `size` | int | `20`（图片列表为 `24`） | 每页条数，超过上限时按上限截断 |
| `cursor` | uint64 | `0` | 游标模式，只返回 `id < cursor` 的记录，此时忽略 `page` |

### 1.4 认证方式

需要登录的接口通过请求头携带凭据：

```
Authorization: Bearer <access_token>
```

支持两类凭据：

| 类型 | 形态 | 说明 |
| --- | --- | --- |
| JWT 访问令牌 | 登录/注册/刷新接口返回的 `access_token` | 默认有效期 2 小时，可访问全部本人接口 |
| API 令牌 | 以 `ph_` 开头（在「API 令牌」页面创建） | 受 `scopes` 限制，明文仅在创建时返回一次 |

API 令牌的权限范围（scope）：

| scope | 作用范围 |
| --- | --- |
| `upload` | `/api/v1/upload/**` 上传相关接口 |
| `read` | `/api/v1/images/**` 图片查询、编辑、删除等接口 |
| `write` | 预留（当前未单独校验） |
| `delete` | 预留（当前未单独校验） |
| `stats` | 预留（当前未单独校验） |
| `*` | 通配，等价于全部权限 |

> 说明：JWT 访问不受 scope 限制；API 令牌缺少所需 scope 时返回 `40300 令牌缺少 xxx 权限`。

访问令牌失效场景：令牌过期返回 `40101`，令牌被注销、用户被禁用或用户不存在均返回 `401/403`。

### 1.5 限流规则

限流基于 Redis 固定窗口实现，命中后返回 `42900` 与 `Retry-After`（秒）响应头。

| 场景 | 维度 | 默认阈值 |
| --- | --- | --- |
| 注册 | IP | 10 次 / 小时 |
| 登录 | IP | 10 次 / 分钟 |
| 上传 | 用户（未登录退化为 IP） | 60 次 / 分钟 |
| 其他需登录接口 | 用户（未登录退化为 IP） | 300 次 / 分钟 |
| 图片直出 `/i` `/t` | IP | 600 次 / 分钟 |

阈值为 `0` 时该维度不限流；Redis 异常时放行请求。

### 1.6 业务错误码

| code | HTTP | 默认消息 | 说明 |
| --- | --- | --- | --- |
| `0` | 200 | `ok` | 成功 |
| `40001` | 400 | 参数错误 | 请求参数缺失或非法 |
| `40010` | 409 | 用户名已占用 | 注册时用户名重复 |
| `40011` | 409 | 邮箱已注册 | 注册时邮箱重复 |
| `40012` | 400 | 用户名不合法 | 长度或字符不符合规则 |
| `40013` | 400 | 密码强度不足 | 未满足密码策略 |
| `40014` | 400 | 验证码错误或已过期 | 验证码校验失败 |
| `40015` | 400 | 邀请码无效 | 邀请码不存在/过期/已用尽 |
| `40020` | 400 | 需要验证码 | 登录失败达阈值后需验证码 |
| `40100` | 401 | 未登录 | 缺少或无效凭据 |
| `40101` | 401 | Token 过期 | 访问/刷新令牌过期 |
| `40300` | 403 | 无权限 | 越权访问、scope 不足、分享密码错误 |
| `40301` | 403 | 邮箱未验证 | 邮箱验证级别为 `force` |
| `40302` | 403 | 请先完成邮箱验证 | 上传等操作要求验证邮箱 |
| `40310` | 403 | S3 签名无效 | S3 网关签名校验失败 |
| `40311` | 403 | S3 时间偏差过大 | 超出 `clock_skew` |
| `40312` | 403 | S3 密钥已禁用 | AccessKey 被禁用 |
| `40400` | 404 | 资源不存在 | 记录不存在或无权访问 |
| `41300` | 413 | 文件超限 | 超过单文件大小限制 |
| `41301` | 413 | 容量配额超限 | 用户总容量不足 |
| `41302` | 413 | 每日上传数超限 | 达到用户组每日上传上限 |
| `42900` | 429 | 请求过于频繁 | 触发限流或登录失败锁定 |
| `50000` | 500 | 内部错误 | 服务端异常 |

### 1.7 通用枚举

| 枚举 | 取值 |
| --- | --- |
| 用户角色 `role` | `0` 普通用户，`1` 管理员 |
| 用户状态 `status` | `0` 禁用，`1` 正常，`2` 待邮箱验证 |
| 通用启用状态 `status` | `0` 禁用，`1` 启用 |
| 图片状态 `status` | `0` 回收站，`1` 正常 |
| 图片可见性 | `public` 公开，`private` 私有 |
| 分享目标 `target_type` | `image` 图片，`album` 相册 |
| 邮件发送状态 `status` | `0` 待发送，`1` 成功，`2` 失败 |
| 邮箱验证级别 | `off` 关闭，`soft` 提示，`force` 强制 |
| 注册模式 | `open` 开放注册，`invite` 邀请码注册 |
| 存储驱动 `driver` | `local`、`s3`、`oss`、`cos`、`webdav` |

---

## 2. 公共接口

### 2.1 存活探针

```
GET /healthz
```

- 鉴权：无
- 成功响应（`200`）：

```json
{ "status": "ok" }
```

### 2.2 就绪探针

```
GET /readyz
```

- 鉴权：无
- 检查数据库与 Redis 可用性
- 成功：`200 {"status":"ready"}`
- 失败：`503 {"status":"unavailable","error":"..."}`

### 2.3 站点公开配置

```
GET /api/v1/site
```

- 鉴权：无
- 用途：前端初始化站点名称、注册开关、验证码开关、上传限制等

响应 `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `name` | string | 站点名称 |
| `description` | string | 站点描述 |
| `logo_url` | string | Logo 地址 |
| `icp` | string | ICP 备案号 |
| `site_url` | string | 站点对外地址 |
| `register_enabled` | bool | 是否开放注册 |
| `register_mode` | string | `open` / `invite` |
| `email_verify_level` | string | `off` / `soft` / `force`，未配置 SMTP 时 `force` 自动降级为 `soft` |
| `captcha_enabled` | bool | 是否启用验证码 |
| `default_visibility` | string | 默认图片可见性 |
| `max_file_size` | int64 | 单文件大小上限（字节） |
| `allowed_extensions` | []string | 允许上传的扩展名（不含点） |
| `thumbnail_enabled` | bool | 是否启用缩略图 |
| `thumbnail_sizes` | []string | 预设缩略图尺寸名称 |
| `s3_gateway_enabled` | bool | S3 网关是否启用 |
| `s3_gateway_prefix` | string | S3 网关路径前缀，默认 `/s3` |

示例：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "name": "PicHub",
    "description": "简洁优雅的图床系统",
    "logo_url": "",
    "icp": "",
    "site_url": "http://localhost:8080",
    "register_enabled": true,
    "register_mode": "open",
    "email_verify_level": "soft",
    "captcha_enabled": true,
    "default_visibility": "public",
    "max_file_size": 10485760,
    "allowed_extensions": ["jpg", "jpeg", "png", "gif", "webp", "bmp", "svg", "avif", "ico"],
    "thumbnail_enabled": true,
    "thumbnail_sizes": ["small", "medium", "large"],
    "s3_gateway_enabled": true,
    "s3_gateway_prefix": "/s3"
  }
}
```

---

## 3. 认证模块

前缀：`/api/v1/auth`

### 3.1 获取图片验证码

```
GET /api/v1/auth/captcha
```

- 鉴权：无
- 说明：验证码未启用时返回 `400 / 40001 验证码未启用`

响应 `data`（`CaptchaResp`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `captcha_id` | string | 验证码 ID，提交时回传 |
| `image_base64` | string | 验证码图片的 Base64（含 data URI 前缀以外内容，按前端约定渲染） |
| `expires_in` | int | 有效期（秒），默认 300 |

### 3.2 注册

```
POST /api/v1/auth/register
```

- 鉴权：无
- 限流：IP 10 次 / 小时
- 请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `username` | string | 是 | 3-32 位，仅 `[a-zA-Z0-9_-]`，不能使用保留名 |
| `email` | string | 是 | 邮箱，接收验证邮件 |
| `password` | string | 是 | 默认至少 8 位且同时包含字母与数字 |
| `captcha_id` | string | 条件 | 验证码启用时必填 |
| `captcha_code` | string | 条件 | 验证码启用时必填 |
| `invite_code` | string | 条件 | 注册模式为 `invite` 时必填 |

业务规则：

- 注册关闭时返回 `40300 注册通道已关闭`
- 同一 IP 24 小时内注册数量受限（默认 3）
- 首个注册用户自动成为管理员，其余为普通用户
- 邮箱验证级别为 `force` 时账号初始状态为 `2`（待验证），需完成邮箱验证后才能登录
- 邮箱验证级别为 `off` 时直接标记为已验证

响应 `data`（`AuthResp`）：见 [16.1 认证响应](#161-认证响应-authresp)

常见错误：`40010`、`40011`、`40012`、`40013`、`40014`、`40015`、`42900`。

### 3.3 登录

```
POST /api/v1/auth/login
```

- 鉴权：无
- 限流：IP 10 次 / 分钟
- 请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `account` | string | 条件 | 用户名或邮箱；与 `username` 二选一 |
| `username` | string | 条件 | 兼容字段，`account` 为空时使用 |
| `password` | string | 是 | 密码 |
| `captcha_id` | string | 条件 | 连续失败达到阈值后必填 |
| `captcha_code` | string | 条件 | 连续失败达到阈值后必填 |

业务规则：

- 同一 IP 连续失败达到 `security.login_max_fails`（默认 5）次后锁定 `security.login_lock_duration`（默认 10 分钟），返回 `42900`
- 连续失败达到 `captcha.login_trigger_fails`（默认 2）次后要求验证码，缺失时返回 `40020`
- 账号被禁用返回 `40300 账号已被禁用`
- 邮箱验证级别为 `force` 且未验证时返回 `40301`

响应 `data`：`AuthResp`

### 3.4 刷新令牌

```
POST /api/v1/auth/refresh
```

- 鉴权：无
- 请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `refresh_token` | string | 是 | 登录时返回的刷新令牌 |

- 说明：刷新令牌会轮换，旧令牌立即失效；过期返回 `40101`

响应 `data`：`AuthResp`

### 3.5 登出

```
POST /api/v1/auth/logout
```

- 鉴权：需要登录
- 请求体（可选）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `refresh_token` | string | 否 | 传入时一并注销该刷新令牌 |

- 响应：`data` 为空

### 3.6 检查用户名可用性

```
GET /api/v1/auth/check-username?u=<用户名>
```

- 鉴权：无
- 参数：`u` 或 `username`
- 响应 `data`（`CheckResp`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `available` | bool | 是否可用 |
| `message` | string | 提示信息 |

### 3.7 检查邮箱可用性

```
GET /api/v1/auth/check-email?e=<邮箱>
```

- 鉴权：无
- 参数：`e` 或 `email`
- 响应 `data`：`CheckResp`

### 3.8 验证邮箱

```
POST /api/v1/auth/verify-email
GET  /api/v1/auth/verify-email?token=<token>
```

- 鉴权：无
- 请求体（POST）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `token` | string | 是 | 邮件中的验证令牌 |

- 说明：验证成功后用户 `email_verified=true`、状态置为正常
- 响应：`data` 为空；令牌无效或过期返回 `40001`

### 3.9 重发验证邮件

```
POST /api/v1/auth/resend-verify
```

- 鉴权：无
- 请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `email` | string | 是 | 邮箱 |
| `captcha_id` | string | 条件 | 验证码启用时必填 |
| `captcha_code` | string | 条件 | 验证码启用时必填 |

- 说明：无论邮箱是否存在均返回成功，避免暴露账号信息

### 3.10 忘记密码

```
POST /api/v1/auth/forgot-password
```

- 鉴权：无
- 请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `email` | string | 是 | 邮箱 |
| `captcha_id` | string | 条件 | 验证码启用时必填 |
| `captcha_code` | string | 条件 | 验证码启用时必填 |

- 说明：SMTP 未启用时返回 `40300 邮件服务未启用，请联系管理员`

### 3.11 重置密码

```
POST /api/v1/auth/reset-password
```

- 鉴权：无
- 请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `token` | string | 是 | 重置邮件中的令牌 |
| `password` | string | 是 | 新密码，需满足密码强度策略 |

- 响应：`data` 为空；令牌无效或过期返回 `40001`

---

## 4. 用户模块

前缀：`/api/v1/user`，均需登录。

### 4.1 获取当前用户资料

```
GET /api/v1/user/profile
```

响应 `data`：`UserResp`，见 [16.2 用户信息](#162-用户信息-userresp)。

### 4.2 修改当前用户资料

```
PUT /api/v1/user/profile
```

请求体（字段均可选，仅传需要修改的字段）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `nickname` | string | 昵称，1-32 字符 |
| `avatar_url` | string | 头像地址，最长 500 |
| `default_visibility` | string | `public` / `private` |
| `referer_stat_enabled` | bool | 是否开启引用统计，配置禁止用户关闭时返回 `40300` |
| `storage_config_id` | uint64 | 绑定本人存储策略，`0` 表示解绑；未开放自定义存储时返回 `40300` |

响应 `data`：更新后的 `UserResp`

### 4.3 修改密码

```
PUT /api/v1/user/password
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `old_password` | string | 是 | 原密码 |
| `new_password` | string | 是 | 新密码，需满足强度策略且不能与原密码相同 |
| `captcha_id` | string | 条件 | 验证码启用时必填 |
| `captcha_code` | string | 条件 | 验证码启用时必填 |

- 响应：`data` 为空

### 4.4 获取用户用量统计

```
GET /api/v1/user/stats
```

响应 `data`（`UserStatsResp`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `image_count` | int64 | 正常图片数量 |
| `album_count` | int64 | 相册数量 |
| `share_count` | int64 | 分享数量 |
| `trash_count` | int64 | 回收站图片数量 |
| `storage_bytes` | int64 | 正常图片占用字节数 |
| `used_bytes` | int64 | 账号已用容量 |
| `quota_bytes` | int64 | 账号总容量 |
| `daily_limit` | int | 每日上传张数上限，`0` 表示不限 |
| `today_uploads` | int | 今日已上传张数 |
| `referer_hits` | int64 | 累计引用命中次数 |
| `top_hosts` | []HostStat | 近 30 天 TOP 引用域名 |
| `trend` | []DayStat | 近 30 天引用趋势 |

`HostStat` 与 `DayStat` 见 [16.9 统计结构](#169-统计结构)。

---

## 5. 上传模块

前缀：`/api/v1/upload`，均需登录，且 API 令牌需具备 `upload` scope。
上传接口按用户限流（默认 60 次 / 分钟）。

### 5.1 上传单张图片

```
POST /api/v1/upload
Content-Type: multipart/form-data
```

表单字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `file` | file | 是 | 图片文件 |
| `album_id` | uint64 | 否 | 归属相册 ID |
| `tags` | string | 否 | 标签，逗号分隔，如 `旅行,风景` |
| `is_public` | bool | 否 | 是否公开，缺省取用户/系统默认可见性 |
| `storage_id` | uint64 | 否 | 指定存储策略 ID，缺省使用默认策略 |

限制与行为：

- 请求体上限为「单文件上限 + 1MB」，超过返回 `41300`
- 校验文件头识别格式，扩展名不在允许列表时返回 `40001`
- 用户级秒传：同用户已存在相同 SHA-256 的正常图片时直接返回，`dedup=true`
- 全局秒传（`upload.dedup_scope=global`）时复用物理文件并累加引用计数
- 达到每日上传上限返回 `41302`，容量不足返回 `41301`
- 上传成功会累加用户已用容量、写入每日上传计数

响应 `data`（`UploadResp`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 图片 ID |
| `url` | string | 图片访问地址 |
| `thumb_url` | string | 缩略图地址，未启用时为空 |
| `markdown` | string | Markdown 引用语法 |
| `html` | string | HTML `<img>` 标签 |
| `bbcode` | string | BBCode 语法 |
| `size` | int64 | 文件大小（字节） |
| `width` | int | 宽（无法解析时为 0） |
| `height` | int | 高（无法解析时为 0） |
| `hash` | string | SHA-256 十六进制摘要 |
| `dedup` | bool | 是否命中秒传 |

### 5.2 秒传检测

```
POST /api/v1/upload/check
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `hash` | string | 是 | 文件 SHA-256，64 位十六进制 |
| `size` | int64 | 否 | 文件大小 |
| `name` | string | 否 | 文件名 |
| `ext` | string | 否 | 扩展名 |

响应 `data`（`UploadCheckResp`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `exists` | bool | 是否已存在可秒传记录 |
| `image` | UploadResp | 命中时返回对应图片信息 |

### 5.3 远程 URL 拉取上传

```
POST /api/v1/upload/url
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `url` | string | 是 | 仅支持 `http` / `https`，禁止内网地址与重定向 |
| `album_id` | uint64 | 否 | 归属相册 |
| `tags` | string | 否 | 逗号分隔标签 |
| `is_public` | bool | 否 | 是否公开 |
| `storage_id` | uint64 | 否 | 指定存储策略 |

- 下载超时 30 秒，响应体超出单文件上限返回 `41300`
- 响应 `data`：`UploadResp`

### 5.4 初始化分片上传

```
POST /api/v1/upload/chunk/init
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | 文件名 |
| `size` | int64 | 是 | 文件总大小，需不超过单文件上限 |
| `hash` | string | 是 | 文件 SHA-256 |
| `chunks` | int | 是 | 分片数量，1-`upload.max_chunk_count`（默认 1000） |
| `album_id` | uint64 | 否 | 归属相册 |
| `tags` | string | 否 | 逗号分隔标签 |
| `is_public` | bool | 否 | 是否公开 |
| `storage_id` | uint64 | 否 | 指定存储策略 |

响应 `data`（`ChunkInitResp`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `upload_id` | string | 分片会话 ID |
| `uploaded` | []int | 已上传分片序号，新建时为空数组 |
| `chunk_size` | int64 | 建议分片大小（字节），默认 5MB |

> 分片会话保存在服务进程内，有效期 24 小时；服务重启后会话失效。

### 5.5 上传单个分片

```
POST /api/v1/upload/chunk/:id
Content-Type: multipart/form-data
```

| 参数 | 位置 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | path | 是 | 分片会话 ID |
| `file` | form | 是 | 分片数据 |
| `index` | form 或 query | 是 | 分片序号，从 0 开始；也接受 `part`、`part_number`、`partNumber` |

- 除最后一片外，单分片不得超过 `chunk_size`；空分片返回 `40001`
- 响应 `data`：

```json
{ "index": 0, "size": 5242880 }
```

### 5.6 完成分片上传

```
POST /api/v1/upload/chunk/:id/complete
```

请求体（可选，服务端以会话中记录的哈希为准）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `hash` | string | 否 | 文件 SHA-256，仅作兼容保留 |

- 校验所有分片是否齐全、合并后大小与哈希是否与初始化时一致
- 响应 `data`：`UploadResp`
- 会话不存在或已过期返回 `40400`

---

## 6. 图片模块

前缀：`/api/v1/images`，均需登录，且 API 令牌需具备 `read` scope。

### 6.1 图片列表

```
GET /api/v1/images
```

查询参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `keyword` | string | 模糊匹配显示名、原始文件名、哈希 |
| `album_id` | uint64 | 按相册筛选 |
| `tag_id` | uint64 | 按标签筛选 |
| `ext` | string | 按扩展名筛选，如 `png`（自动去点、转小写） |
| `referred` | bool | 仅返回存在引用记录的图片 |
| `is_public` | bool | 按可见性筛选 |
| `trash` | bool | 为 `true` 时查询回收站 |
| `cursor` | uint64 | 游标模式，只返回 `id < cursor` 的记录 |
| `page` | int | 页码，默认 1 |
| `size` | int | 每页条数，默认 24，最大 200 |
| `with_tags` | string | `1` 或 `true` 时为每张图片附带标签列表 |

排序规则：正常图片按 `created_at DESC, id DESC`；回收站按 `deleted_at DESC, id DESC`。

响应 `data`：`PageResp`，`list` 为 `ImageResp[]`，见 [16.3 图片信息](#163-图片信息-imageresp)。

### 6.2 回收站列表

```
GET /api/v1/images/trash
```

- 参数同 6.1（`trash` 固定为 `true`）
- 响应 `data`：`PageResp`，`list` 为 `ImageResp[]`，包含 `deleted_at`

### 6.3 清空回收站

```
DELETE /api/v1/images/trash
```

- 说明：逐张永久删除回收站中的图片（含关联引用、分享、标签、存储对象）
- 响应 `data`：

```json
{ "affected": 12 }
```

### 6.4 批量操作

```
POST /api/v1/images/batch
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `ids` | []uint64 | 是 | 图片 ID 列表，单次最多 500 个 |
| `action` | string | 是 | `delete` / `restore` / `move` / `public` / `private` |
| `album_id` | uint64 | 条件 | `action=move` 时必填，为目标相册 ID |

动作说明：

| action | 说明 |
| --- | --- |
| `delete` | 移入回收站（仅处理正常图片） |
| `restore` | 从回收站还原（仅处理回收站图片） |
| `move` | 移动到指定相册 |
| `public` | 批量设为公开 |
| `private` | 批量设为私有 |

响应 `data`：

```json
{ "affected": 8 }
```

### 6.5 获取图片详情

```
GET /api/v1/images/:id
```

- 响应 `data`：`ImageResp`（包含 `tags`）

### 6.6 修改图片信息

```
PATCH /api/v1/images/:id
```

请求体（字段均可选）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `display_name` | string | 显示名称，1-255 字符 |
| `album_id` | uint64 | 目标相册，`0` 表示移出相册 |
| `clear_album` | bool | 为 `true` 时移出相册，优先级高于 `album_id` |
| `tags` | []string | 覆盖式同步标签，传空数组表示清空 |
| `is_public` | bool | 是否公开 |

- 响应 `data`：更新后的 `ImageResp`

### 6.7 移入回收站

```
DELETE /api/v1/images/:id
```

- 说明：软删除，图片状态置为 `0`，写入 `deleted_at`
- 响应：`data` 为空

### 6.8 永久删除

```
DELETE /api/v1/images/:id/permanent
```

- 说明：删除数据库记录、引用记录、图片分享、标签关联，并在存储 Key 不再被引用时删除物理文件
- 响应：`data` 为空

### 6.9 从回收站还原

```
POST /api/v1/images/:id/restore
```

- 响应：`data` 为空；图片不在回收站时也返回成功

### 6.10 图片引用来源列表

```
GET /api/v1/images/:id/referers
```

查询参数：

| 参数 | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `page` | int | 1 | 页码 |
| `size` | int | 20 | 每页条数，最大 200 |
| `order` | string | `hits` | `hits` 按命中次数倒序；`time` 按最近访问时间倒序 |

响应 `data`：`PageResp`，`list` 为 `RefererResp[]`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 引用记录 ID |
| `host` | string | 来源域名 |
| `url` | string | 最近一次来源地址 |
| `hits` | int64 | 命中次数 |
| `first_seen_at` | time | 首次出现时间 |
| `last_seen_at` | time | 最近出现时间 |

### 6.11 图片引用汇总

```
GET /api/v1/images/:id/referers/summary
```

响应 `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `total_hits` | int64 | 总命中次数 |
| `host_count` | int64 | 引用域名数量 |

### 6.12 删除单条引用记录

```
DELETE /api/v1/images/:id/referers/:rid
```

- 响应：`data` 为空

---

## 7. 相册模块

前缀：`/api/v1/albums`，均需登录。

### 7.1 相册列表

```
GET /api/v1/albums
```

响应 `data`：

```json
{ "list": [ /* AlbumResp */ ] }
```

`AlbumResp` 字段见 [16.4 相册与标签](#164-相册与标签)。

### 7.2 创建相册

```
POST /api/v1/albums
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | 相册名称 |
| `description` | string | 否 | 描述 |
| `is_public` | bool | 否 | 是否公开，缺省为私有 |

- 响应 `data`：`AlbumResp`

### 7.3 获取相册详情

```
GET /api/v1/albums/:id
```

- 响应 `data`：`AlbumResp`

### 7.4 更新相册

```
PUT /api/v1/albums/:id
```

请求体（字段均可选）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `name` | string | 相册名称 |
| `description` | string | 描述 |
| `is_public` | bool | 是否公开 |
| `cover_image_id` | uint64 | 封面图片 ID |
| `sort_order` | int | 排序值 |

- 响应 `data`：`AlbumResp`

### 7.5 删除相册

```
DELETE /api/v1/albums/:id
```

- 说明：仅删除相册本身，图片保留
- 响应：`data` 为空

---

## 8. 标签模块

前缀：`/api/v1/tags`，均需登录。

### 8.1 标签列表

```
GET /api/v1/tags
```

响应 `data`：

```json
{ "list": [ /* TagResp */ ] }
```

### 8.2 创建标签

```
POST /api/v1/tags
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | 标签名称 |
| `color` | string | 否 | 颜色值，如 `#409EFF` |

- 响应 `data`：`TagResp`

### 8.3 更新标签

```
PUT /api/v1/tags/:id
```

请求体（字段均可选）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `name` | string | 标签名称 |
| `color` | string | 颜色值 |

- 响应 `data`：`TagResp`

### 8.4 删除标签

```
DELETE /api/v1/tags/:id
```

- 响应：`data` 为空

---

## 9. API 令牌与 S3 凭据

### 9.1 API 令牌列表

```
GET /api/v1/tokens
```

- 鉴权：需要登录
- 响应 `data`：

```json
{ "list": [ /* TokenResp */ ] }
```

### 9.2 创建 API 令牌

```
POST /api/v1/tokens
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | 令牌名称，1-64 字符 |
| `scopes` | []string | 否 | 可选 `upload`、`read`、`write`、`delete`、`stats`；为空或缺省时授予全部；非法值被忽略 |
| `expires_in_days` | int | 否 | 有效天数，`0` 表示永不过期，最大 3650 |

- 受用户组 `max_tokens` 限制，超限返回 `40300 令牌数量已达上限`
- 响应 `data`（`TokenCreateResp`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `token` | string | 明文令牌，形如 `ph_xxx`，**仅返回一次** |
| `info` | TokenResp | 令牌信息 |

### 9.3 删除 API 令牌

```
DELETE /api/v1/tokens/:id
```

- 响应：`data` 为空

### 9.4 启用 / 禁用 API 令牌

```
PUT /api/v1/tokens/:id/status
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `status` | int8 | 是 | `0` 禁用，`1` 启用 |

- 响应：`data` 为空

### 9.5 S3 凭据列表

```
GET /api/v1/s3/credentials
```

- 鉴权：需要登录
- 响应 `data`：

```json
{ "list": [ /* S3CredResp */ ] }
```

### 9.6 创建 S3 凭据

```
POST /api/v1/s3/credentials
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | 凭据名称，1-64 字符 |

- 响应 `data`（`S3CredCreateResp`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `access_key` | string | AccessKey |
| `secret_key` | string | SecretKey，**仅返回一次** |
| `info` | S3CredResp | 凭据信息 |

### 9.7 删除 S3 凭据

```
DELETE /api/v1/s3/credentials/:id
```

- 响应：`data` 为空

---

## 10. 分享模块

### 10.1 我的分享列表

```
GET /api/v1/shares
```

- 鉴权：需要登录
- 响应 `data`：

```json
{ "list": [ /* ShareResp */ ] }
```

### 10.2 创建分享

```
POST /api/v1/shares
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `target_type` | string | 是 | `image` 或 `album` |
| `target_id` | uint64 | 是 | 目标图片/相册 ID，必须属于当前用户 |
| `password` | string | 否 | 访问密码，4-32 位；不传表示无密码 |
| `expires_in_hours` | int | 否 | 有效期（小时），`0` 表示永久，最大 8760 |
| `max_views` | int | 否 | 最大访问次数，`0` 表示不限 |

- 受用户组 `max_shares` 限制，超限返回 `40300 分享数量已达上限`
- 分享链接形如 `<site_url>/s/<slug>`
- 响应 `data`：`ShareResp`

### 10.3 删除分享

```
DELETE /api/v1/shares/:id
```

- 响应：`data` 为空

### 10.4 访问分享（公开）

```
GET /api/v1/s/:slug?password=<密码>
```

- 鉴权：无（该接口位于 `/api/v1` 下，不走登录校验）
- 参数：`password` 也可通过表单 `password` 提交
- 校验顺序：分享存在且启用 → 未过期 → 未超出访问次数 → 密码正确
- 成功会增加访问次数
- 响应 `data`（`ShareAccessResp`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `share` | ShareResp | 分享信息 |
| `album` | AlbumResp | `target_type=album` 时返回 |
| `images` | []ImageResp | 图片分享返回 1 张；相册分享最多返回 200 张 |

常见错误：`40400`（不存在/已失效/已过期/次数用尽）、`40300`（访问密码错误）。

---

## 11. 存储策略模块

前缀：`/api/v1/storages`，均需登录。此处为「当前用户私有存储策略」，
系统级存储策略由管理端接口管理。

### 11.1 存储策略列表

```
GET /api/v1/storages
```

响应 `data`：

```json
{ "list": [ /* StorageResp */ ] }
```

### 11.2 创建存储策略

```
POST /api/v1/storages
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | 策略名称 |
| `driver` | string | 是 | `local` / `s3` / `oss` / `cos` / `webdav` |
| `domain` | string | 否 | 自定义访问域名 |
| `path_prefix` | string | 否 | 对象 Key 路径前缀 |
| `is_default` | bool | 否 | 是否设为默认策略 |
| `status` | int8 | 否 | `0` 禁用，`1` 启用，缺省启用 |
| `root_path` | string | 否 | 本地存储根目录（`local`） |
| `path_template` | string | 否 | 对象 Key 模板，如 `{prefix}/{yyyy}/{MM}/{dd}/{hash}.{ext}` |
| `endpoint` | string | 否 | S3/OSS/COS 端点 |
| `region` | string | 否 | 区域 |
| `bucket` | string | 否 | Bucket |
| `path_style` | bool | 否 | 是否使用 Path Style 访问 |
| `use_ssl` | bool | 否 | 是否使用 HTTPS |
| `public_url` | string | 否 | 公共访问地址前缀 |
| `url` | string | 否 | WebDAV 地址 |
| `access_key` | string | 否 | S3/OSS/COS AccessKey |
| `secret_key` | string | 否 | S3/OSS/COS SecretKey |
| `username` | string | 否 | WebDAV 用户名 |
| `password` | string | 否 | WebDAV 密码 |

- 响应 `data`：`StorageResp`（不返回密钥明文，仅 `has_secret` 标识）

### 11.3 更新存储策略

```
PUT /api/v1/storages/:id
```

- 请求体字段同创建接口，全部可选（仅传需要修改的字段）
- 响应 `data`：`StorageResp`

### 11.4 删除存储策略

```
DELETE /api/v1/storages/:id
```

- 响应：`data` 为空

### 11.5 测试存储策略连通性

```
POST /api/v1/storages/:id/test
```

- 响应 `data`（`StorageTestResp`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ok` | bool | 是否连通 |
| `message` | string | 结果描述 |

---

## 12. 引用统计模块

前缀：`/api/v1/stats`，均需登录。

### 12.1 TOP 引用域名

```
GET /api/v1/stats/referers/top?limit=10&days=30
```

| 参数 | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `limit` | int | 10 | 返回条数 |
| `days` | int | 30 | 统计最近天数 |

响应 `data`：

```json
{
  "list": [
    { "host": "example.com", "hits": 128, "images": 6 }
  ]
}
```

### 12.2 引用趋势

```
GET /api/v1/stats/referers/trend?days=30
```

| 参数 | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `days` | int | 30 | 统计最近天数 |

响应 `data`：

```json
{
  "list": [
    { "date": "2026-10-24", "hits": 64 }
  ]
}
```

---

## 13. 管理端模块

前缀：`/api/v1/admin`，需登录且 `role=1`（管理员），API 令牌同样需要管理员所属账号。

### 13.1 仪表盘

```
GET /api/v1/admin/dashboard
```

响应 `data`（`DashboardResp`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `user_count` | int64 | 用户总数 |
| `image_count` | int64 | 正常图片总数 |
| `storage_bytes` | int64 | 全站存储占用 |
| `today_uploads` | int64 | 今日上传数 |
| `referer_hits` | int64 | 全站引用命中次数 |
| `album_count` | int64 | 相册总数 |
| `share_count` | int64 | 分享总数 |
| `trash_count` | int64 | 回收站图片数 |
| `group_count` | int64 | 用户组数量 |
| `invite_count` | int64 | 邀请码数量 |
| `queue_ready` | int64 | 待处理队列任务数 |
| `queue_delayed` | int64 | 延迟队列任务数 |
| `queue_dead` | int64 | 死信队列任务数 |

### 13.2 用户列表

```
GET /api/v1/admin/users?keyword=&status=&role=&group_id=&page=1&size=20
```

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `keyword` | string | 匹配用户名/邮箱/昵称 |
| `status` | int8 | 用户状态筛选 |
| `role` | int8 | 角色筛选 |
| `group_id` | uint64 | 用户组筛选 |
| `page` / `size` | int | 分页，默认 1 / 20 |

响应 `data`：`PageResp`，`list` 为 `AdminUserResp[]`（含 `register_ip`、`last_login_ip` 等管理字段）。

### 13.3 启用 / 禁用用户

```
PUT /api/v1/admin/users/:id/status
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `status` | int8 | 是 | `0` 禁用，`1` 正常，`2` 待验证 |
| `reason` | string | 否 | 操作原因，写入审计日志 |

- 响应：`data` 为空

### 13.4 调整用户组与配额

```
PUT /api/v1/admin/users/:id/group
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `group_id` | uint64 | 是 | 目标用户组 |
| `quota_bytes` | int64 | 否 | 覆盖总容量 |
| `max_file_size` | int64 | 否 | 覆盖单文件上限 |
| `daily_limit` | int | 否 | 覆盖每日上传上限 |

- 响应：`data` 为空

### 13.5 重置用户密码

```
PUT /api/v1/admin/users/:id/password
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `new_password` | string | 是 | 新密码，需满足密码强度策略 |

- 响应：`data` 为空

### 13.6 用户组列表

```
GET /api/v1/admin/groups
```

响应 `data`：

```json
{ "list": [ /* GroupResp */ ] }
```

### 13.7 创建用户组

```
POST /api/v1/admin/groups
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | 名称，1-32 字符 |
| `description` | string | 否 | 描述 |
| `quota_bytes` | int64 | 否 | 总容量（字节），`0` 表示不限 |
| `max_file_size` | int64 | 否 | 单文件上限 |
| `daily_limit` | int | 否 | 每日上传上限 |
| `max_albums` | int | 否 | 相册数量上限 |
| `max_tokens` | int | 否 | 令牌数量上限 |
| `max_shares` | int | 否 | 分享数量上限 |
| `allow_register` | bool | 否 | 是否允许注册到该组 |
| `allow_custom_storage` | bool | 否 | 是否允许自定义存储 |
| `is_default` | bool | 否 | 是否设为默认组（会清除其它默认组） |
| `status` | int8 | 否 | 启用状态，缺省启用 |

- 响应 `data`：`GroupResp`

### 13.8 更新用户组

```
PUT /api/v1/admin/groups/:id
```

- 请求体字段同创建接口，全部可选
- 响应 `data`：`GroupResp`

### 13.9 删除用户组

```
DELETE /api/v1/admin/groups/:id
```

- 默认用户组或组下仍有用户时返回 `40300`
- 响应：`data` 为空

### 13.10 系统设置列表

```
GET /api/v1/admin/settings
```

响应 `data`：

```json
{ "list": [ /* SettingResp */ ] }
```

- 密文配置项的 `value` 返回掩码 `******`

### 13.11 批量更新系统设置

```
PUT /api/v1/admin/settings
```

请求体：

```json
{
  "settings": {
    "site.name": "PicHub",
    "register.enabled": "true"
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `settings` | map[string]string | 是 | 配置键值对，值统一以字符串提交 |

规则：

- 不存在的配置键返回 `40001 未知配置项: <key>`
- 密文配置传 `******` 表示保持原值
- 更新后立即失效设置缓存

常用配置键：

| 键 | 说明 |
| --- | --- |
| `site.name` | 站点名称 |
| `site.url` | 站点对外地址 |
| `register.enabled` | 是否开放注册 |
| `register.mode` | `open` / `invite` |
| `register.require_email_verify` | `off` / `soft` / `force` |
| `register.ip_limit_per_day` | 单 IP 每日注册上限 |
| `register.default_group_id` | 注册默认用户组 ID |
| `captcha.enabled` | 是否启用验证码 |
| `captcha.login_trigger_fails` | 登录失败几次后要求验证码 |
| `upload.default_visibility` | 默认可见性 |
| `upload.allowed_extensions` | 允许的扩展名，JSON 数组或逗号分隔 |
| `referer.enabled` | 是否启用引用统计 |
| `referer.max_per_image` | 单图引用来源上限 |
| `s3_gateway.enabled` | 是否启用 S3 网关 |
| `s3_gateway.path_prefix` | S3 网关路径前缀 |

- 响应：`data` 为空

### 13.12 操作日志

```
GET /api/v1/admin/logs?user_id=&action=&page=1&size=20
```

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `user_id` | uint64 | 操作者用户 ID |
| `action` | string | 动作标识，如 `user.login`、`admin.group_create` |
| `page` / `size` | int | 分页，默认 1 / 20 |

响应 `data`：`PageResp`，`list` 为 `LogResp[]`（含 `ip`、`user_agent`、`detail`）。

### 13.13 邮件日志

```
GET /api/v1/admin/email-logs?user_id=&status=&page=1&size=20
```

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `user_id` | uint64 | 关联用户 ID |
| `status` | int8 | `0` 待发送，`1` 成功，`2` 失败 |
| `page` / `size` | int | 分页，默认 1 / 20 |

响应 `data`：`PageResp`，`list` 为 `EmailLogResp[]`。

### 13.14 邀请码列表

```
GET /api/v1/admin/invites?page=1&size=20
```

响应 `data`：`PageResp`，`list` 为 `InviteResp[]`。

### 13.15 批量创建邀请码

```
POST /api/v1/admin/invites
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `max_uses` | int | 否 | 单个邀请码可用次数，`0` 表示不限 |
| `group_id` | uint64 | 否 | 注册后归属用户组 |
| `expires_in_hours` | int | 否 | 有效期（小时），`0` 表示永久 |
| `count` | int | 否 | 生成数量，默认 1 |

响应 `data`：

```json
{ "list": [ /* InviteResp */ ] }
```

### 13.16 删除邀请码

```
DELETE /api/v1/admin/invites/:id
```

- 响应：`data` 为空

### 13.17 系统存储策略列表

```
GET /api/v1/admin/storages
```

- 响应 `data`：`{ "list": [ /* StorageResp */ ] }`（含全部系统级策略）

### 13.18 系统存储策略增删改查

```
POST   /api/v1/admin/storages
PUT    /api/v1/admin/storages/:id
DELETE /api/v1/admin/storages/:id
POST   /api/v1/admin/storages/:id/test
```

- 请求体与用户存储策略接口一致（见 [11.2](#112-创建存储策略)），创建的策略属于系统级（`user_id` 为空）
- 响应：创建/更新返回 `StorageResp`，删除返回空数据，测试返回 `StorageTestResp`

---

## 14. 图片直出

图片直出接口不返回 JSON，直接输出图片二进制，按 IP 限流（默认 600 次 / 分钟）。

### 14.1 原图与动态处理

```
GET /i/:hash
GET /i/:hash?w=800&h=600&fit=cover&q=85&format=webp
```

路径参数：

| 参数 | 说明 |
| --- | --- |
| `hash` | 图片 SHA-256；可带图片扩展名（如 `/i/<hash>.png`），服务端会自动剥离 |

动态处理查询参数（任一参数存在且需要处理时走动态处理链路）：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `w` | int | 目标宽度，超过 `image.dynamic.max_width`（默认 4096）时截断 |
| `h` | int | 目标高度，超过 `image.dynamic.max_height`（默认 4096）时截断 |
| `fit` | string | `cover` / `contain` / `fill` / `inside` / `outside`，默认 `inside` |
| `q` | int | 输出质量 1-100，默认 `image.dynamic.default_quality`（85） |
| `format` | string | 输出格式，需在 `image.dynamic.allowed_formats`（默认 `webp,jpeg,png,avif`）内 |
| `blur` | float | 模糊程度 0-100，内部换算为 sigma（0.3-20） |
| `gray` | bool | `1` / `true` / `yes` 时输出灰度图 |

处理约束：

- 默认不放大（`no_upscale`）
- 源图解码像素超过 1 亿或源文件超过 256MB 返回 `41300`
- SVG 不支持动态处理，返回 `40001 该图片格式不支持处理`
- 处理结果按「源图 + 参数」磁盘缓存，缓存键版本固定

私有图片访问签名参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `uid` | uint64 | 图片所属用户 ID，与图片归属一致时视为本人访问 |
| `exp` | int64 | 签名过期时间戳（Unix 秒） |
| `sign` | string | 签名值，`hex(HMAC-SHA256(secret, hash + ":" + exp))` |

- `secret` 取 `security.url_sign_secret`，为空时回退 `security.jwt_secret`
- 公开图片无需签名；私有图片非本人访问且签名缺失/过期/不匹配时返回 `40300 图片不可访问`

响应头：

| 响应头 | 说明 |
| --- | --- |
| `ETag` | 原图为 `"<hash>"`；处理图为 `"<hash>\|<参数键>"` |
| `Cache-Control` | 公开原图 `public, max-age=31536000, immutable`；私有原图 `private, max-age=3600`；处理图 `public, max-age=604800` |
| `X-Content-Type-Options` | 固定 `nosniff` |
| `If-None-Match` | 命中时返回 `304 Not Modified` |

防盗链：`image.anti_leech.enabled=true` 时校验 `Referer`

- 空 Referer：`allow_empty_referer=true` 放行，否则拒绝
- 同站点域名或其子域放行；命中 `referer_whitelist` 放行
- 拒绝行为由 `block_action` 决定：`forbidden` 返回 `40300`，其他值（默认 `redirect`）返回 `302` 跳转到站点首页

### 14.2 预设缩略图

```
GET /t/:hash/:size
```

| 参数 | 说明 |
| --- | --- |
| `hash` | 图片 SHA-256，可带扩展名 |
| `size` | 预设尺寸名（默认 `small` / `medium` / `large`）或像素宽度（如 `800`），可带扩展名 |

- 缩略图未启用返回 `40400`，未知尺寸返回 `40001 不支持的缩略图尺寸`
- 输出格式取 `image.thumbnail.format`（默认 `webp`）
- 响应头同动态处理（`ETag`、`Cache-Control: public, max-age=604800`）

---

## 15. S3 兼容网关

网关前缀由 `s3_gateway.path_prefix` 决定，默认 `/s3`；仅在 `s3_gateway.enabled=true` 时可用，
关闭时返回 `503 ServiceUnavailable`。

### 15.1 连接参数

| 项目 | 值 |
| --- | --- |
| Endpoint | `<site_url>/s3` |
| Region | `s3_gateway.region`，默认 `us-east-1` |
| Bucket | 逻辑 Bucket 固定为 `pichub`（列举接口返回），对象 Key 即存储 Key |
| 鉴权 | AWS Signature V4（`Authorization` 头或 `X-Amz-Signature` 查询参数） |
| 凭据 | 在「API 令牌 → S3 凭据」创建，SecretKey 仅显示一次 |
| Path Style | 必须开启 |
| 时钟偏差 | 默认允许 15 分钟 |
| 防重放 | 默认 15 分钟窗口内同一签名不可重复使用 |

Bucket 名称校验：3-63 位，仅小写字母、数字、`-`、`.`，且首尾不能为符号。

### 15.2 支持的接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/s3` | ListBuckets，返回 `pichub` |
| `GET` | `/s3/:bucket?location` | GetBucketLocation，返回 Region |
| `HEAD` | `/s3/:bucket` | HeadBucket，返回 `x-amz-bucket-region` |
| `GET` | `/s3/:bucket` | ListObjectsV2，支持 `prefix`、`delimiter`、`max-keys`（最大 1000）、`continuation-token` |
| `PUT` | `/s3/:bucket/:key` | PutObject，响应头 `ETag` 为内容 MD5 |
| `GET` | `/s3/:bucket/:key` | GetObject，支持 `Range` 断点续传 |
| `HEAD` | `/s3/:bucket/:key` | HeadObject，返回 `Content-Length`、`ETag`、`Content-Type` |
| `DELETE` | `/s3/:bucket/:key` | DeleteObject，成功返回 `204`（对象不存在同样返回 `204`） |
| `POST` | `/s3/:bucket/:key?uploads` | CreateMultipartUpload，需 `allow_multipart=true` |
| `PUT` | `/s3/:bucket/:key?uploadId=&partNumber=` | UploadPart，`partNumber` 取值 1-10000 |
| `POST` | `/s3/:bucket/:key?uploadId=` | CompleteMultipartUpload，请求体为标准 XML |
| `DELETE` | `/s3/:bucket/:key?uploadId=` | AbortMultipartUpload，返回 `204` |

行为说明：

- `PutObject` 会校验请求体 `x-amz-content-sha256`（若提供），并按用户单文件上限限制大小
- 对象写入复用图片表，内容相同的 Key 会做记录迁移；相同内容重复上传视为秒传
- 删除对象等于把对应图片移入回收站
- 匿名 `GET` / `HEAD`（无 `Authorization` 且无 `X-Amz-Signature`）仅允许访问公开图片，便于 PicGo 自定义域名直链
- 对象直链：`<site_url>/s3/pichub/<key>`

### 15.3 错误响应

S3 相关错误以 XML 返回，`Content-Type: application/xml`：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Error>
  <Code>NoSuchKey</Code>
  <Message>对象不存在</Message>
  <RequestId>6f1c0f9a3d2b7e54</RequestId>
  <HostId>PicHub</HostId>
</Error>
```

常见错误码：

| Code | HTTP | 说明 |
| --- | --- | --- |
| `AccessDenied` | 403 | 无权限、密钥禁用或账号不可用 |
| `SignatureDoesNotMatch` | 403 | 签名校验失败 |
| `RequestTimeTooSkewed` | 403 | 请求时间偏差过大 |
| `RequestReplay` | 403 | 签名被重放 |
| `NoSuchKey` | 404 | 对象不存在 |
| `NoSuchUpload` | 404 | 分片上传会话不存在或已过期 |
| `InvalidBucketName` | 400 | Bucket 名称不合法 |
| `InvalidRequest` | 400 | 参数不合法 |
| `MalformedXML` | 400 | 完成分片的 XML 不合法 |
| `XAmzContentSHA256Mismatch` | 400 | 请求体哈希与声明不一致 |
| `EntityTooLarge` | 413 | 对象大小超限 |
| `QuotaExceeded` | 403 | 容量配额不足 |
| `SlowDown` | 503 | 每日上传数超限 |
| `MethodNotAllowed` | 405 | 不支持的方法 |
| `NotImplemented` | 501 | 未开启分片上传 |
| `ServiceUnavailable` | 503 | S3 网关已关闭 |

---

## 16. 附录：数据结构

### 16.1 认证响应 AuthResp

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `access_token` | string | JWT 访问令牌 |
| `refresh_token` | string | 刷新令牌 |
| `expires_in` | int64 | 访问令牌有效期（秒） |
| `user` | UserResp | 当前用户信息 |

### 16.2 用户信息 UserResp

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 用户 ID |
| `username` | string | 用户名 |
| `email` | string | 邮箱 |
| `nickname` | string | 昵称 |
| `avatar_url` | string | 头像地址 |
| `role` | int8 | `0` 用户，`1` 管理员 |
| `status` | int8 | 用户状态 |
| `email_verified` | bool | 邮箱是否已验证 |
| `group_id` | uint64 | 用户组 ID |
| `quota_bytes` | int64 | 总容量（字节） |
| `used_bytes` | int64 | 已用容量（字节） |
| `max_file_size` | int64 | 单文件上限（字节） |
| `daily_limit` | int | 每日上传上限，`0` 不限 |
| `default_visibility` | string | 默认可见性 |
| `storage_config_id` | uint64\|null | 绑定的存储策略 ID |
| `referer_stat_enabled` | bool | 是否开启引用统计 |
| `last_login_at` | time\|null | 最近登录时间 |
| `created_at` | time | 注册时间 |

### 16.3 图片信息 ImageResp

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 图片 ID |
| `hash` | string | SHA-256 |
| `name` | string | 显示名称 |
| `origin_name` | string | 原始文件名 |
| `ext` | string | 扩展名（不含点） |
| `mime` | string | MIME 类型 |
| `size` | int64 | 字节数 |
| `width` | int | 宽度 |
| `height` | int | 高度 |
| `url` | string | 访问地址（本地存储由站点地址派生） |
| `thumb_url` | string | 缩略图地址，未启用时为空 |
| `is_public` | bool | 是否公开 |
| `album_id` | uint64\|null | 所属相册 |
| `tags` | []TagResp | 标签列表，仅详情或 `with_tags` 时返回 |
| `ref_count` | int | 物理文件被引用次数 |
| `status` | int8 | `0` 回收站，`1` 正常 |
| `created_at` | time | 上传时间 |
| `deleted_at` | time\|null | 移入回收站时间 |

### 16.4 相册与标签

`AlbumResp`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 相册 ID |
| `name` | string | 名称 |
| `description` | string | 描述 |
| `cover_image_id` | uint64\|null | 封面图片 ID |
| `cover_url` | string | 封面地址（预留字段，当前恒为空字符串） |
| `is_public` | bool | 是否公开 |
| `sort_order` | int | 排序值 |
| `image_count` | int | 图片数量 |
| `created_at` | time | 创建时间 |
| `updated_at` | time | 更新时间 |

`TagResp`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 标签 ID |
| `name` | string | 名称 |
| `color` | string | 颜色 |
| `image_count` | int | 关联图片数 |
| `created_at` | time | 创建时间 |

### 16.5 令牌与凭据

`TokenResp`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 令牌 ID |
| `name` | string | 名称 |
| `prefix` | string | 令牌前缀（用于辨识） |
| `scopes` | []string | 权限范围 |
| `expires_at` | time\|null | 过期时间，`null` 表示永久 |
| `last_used_at` | time\|null | 最近使用时间 |
| `call_count` | int64 | 调用次数 |
| `status` | int8 | `0` 禁用，`1` 启用 |
| `created_at` | time | 创建时间 |

`S3CredResp`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 凭据 ID |
| `name` | string | 名称 |
| `access_key` | string | AccessKey |
| `call_count` | int64 | 调用次数 |
| `last_used_at` | time\|null | 最近使用时间 |
| `status` | int8 | `0` 禁用，`1` 启用 |
| `created_at` | time | 创建时间 |

### 16.6 分享信息 ShareResp

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 分享 ID |
| `slug` | string | 短链接标识 |
| `target_type` | string | `image` / `album` |
| `target_id` | uint64 | 目标 ID |
| `has_password` | bool | 是否设置访问密码 |
| `expires_at` | time\|null | 过期时间 |
| `max_views` | int | 最大访问次数，`0` 不限 |
| `views` | int | 已访问次数 |
| `status` | int8 | `0` 失效，`1` 正常 |
| `url` | string | 分享完整地址 |
| `created_at` | time | 创建时间 |

### 16.7 存储策略 StorageResp

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 策略 ID |
| `user_id` | uint64\|null | 所属用户，`null` 表示系统级 |
| `name` | string | 名称 |
| `driver` | string | 驱动类型 |
| `domain` | string | 自定义域名 |
| `path_prefix` | string | 路径前缀 |
| `is_default` | bool | 是否默认策略 |
| `status` | int8 | 启用状态 |
| `root_path` | string | 本地根目录（`local`） |
| `path_template` | string | 对象 Key 模板 |
| `endpoint` | string | 服务端点 |
| `region` | string | 区域 |
| `bucket` | string | Bucket |
| `path_style` | bool | 是否 Path Style |
| `use_ssl` | bool\|null | 是否 HTTPS |
| `public_url` | string | 公共访问前缀 |
| `url` | string | WebDAV 地址 |
| `has_secret` | bool | 是否已配置密钥 |
| `last_test_at` | time\|null | 最近连通性测试时间 |
| `last_test_ok` | bool\|null | 最近测试结果 |
| `created_at` | time | 创建时间 |
| `updated_at` | time | 更新时间 |

### 16.8 管理端结构

`AdminUserResp`：在 `UserResp` 基础上增加 `register_ip`、`last_login_ip`，不含 `default_visibility`、`storage_config_id`。

`GroupResp`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 用户组 ID |
| `name` | string | 名称 |
| `description` | string | 描述 |
| `quota_bytes` | int64 | 总容量 |
| `max_file_size` | int64 | 单文件上限 |
| `daily_limit` | int | 每日上传上限 |
| `max_albums` | int | 相册上限 |
| `max_tokens` | int | 令牌上限 |
| `max_shares` | int | 分享上限 |
| `allow_register` | bool | 是否允许注册到该组 |
| `allow_custom_storage` | bool | 是否允许自定义存储 |
| `is_default` | bool | 是否默认组 |
| `status` | int8 | 启用状态 |
| `user_count` | int64 | 组内用户数 |
| `created_at` | time | 创建时间 |
| `updated_at` | time | 更新时间 |

`SettingResp`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `key` | string | 配置键 |
| `value` | string | 配置值，密文项为 `******` |
| `type` | string | 值类型，如 `string` / `int` / `bool` / `json` |
| `is_secret` | bool | 是否密文项 |
| `description` | string | 说明 |
| `updated_at` | time | 更新时间 |

`LogResp`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 日志 ID |
| `user_id` | uint64 | 操作者 |
| `action` | string | 动作标识 |
| `target_type` | string | 目标类型 |
| `target_id` | uint64 | 目标 ID |
| `ip` | string | 客户端 IP |
| `user_agent` | string | User-Agent |
| `detail` | object | 附加信息（JSON） |
| `created_at` | time | 发生时间 |

`InviteResp`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 邀请码 ID |
| `code` | string | 邀请码 |
| `creator_id` | uint64 | 创建者 |
| `max_uses` | int | 最大使用次数 |
| `used_count` | int | 已使用次数 |
| `group_id` | uint64 | 注册后归属用户组 |
| `expires_at` | time\|null | 过期时间 |
| `status` | int8 | 启用状态 |
| `created_at` | time | 创建时间 |

`EmailLogResp`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 日志 ID |
| `user_id` | uint64 | 关联用户 |
| `to_email` | string | 收件邮箱 |
| `subject` | string | 主题 |
| `template` | string | 模板标识 |
| `status` | int8 | `0` 待发送，`1` 成功，`2` 失败 |
| `error` | string | 失败原因 |
| `retry` | int8 | 重试次数 |
| `sent_at` | time\|null | 发送时间 |
| `created_at` | time | 创建时间 |

### 16.9 统计结构

`HostStat`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `host` | string | 来源域名 |
| `hits` | int64 | 命中次数 |
| `images` | int64 | 涉及图片数 |

`DayStat`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `date` | string | 日期，格式 `YYYY-MM-DD` |
| `hits` | int64 | 当日命中次数 |
