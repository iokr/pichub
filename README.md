# PicHub

PicHub 是一个自托管的图床系统。单个 Go 二进制内嵌 Vue 3 前端，同时提供 Web 界面、REST API、图片直出与 S3 兼容网关，默认使用本地磁盘存储，也可切换到 S3 / OSS / COS / WebDAV。

## 功能特性

- **上传**：拖拽、粘贴、点击上传，多文件并发（默认 3），进度、取消、重试
- **秒传**：基于 SHA-256 去重，支持用户级 / 全站级去重策略
- **图库**：关键字、相册、标签、可见性筛选，批量移动 / 公开 / 私有 / 删除
- **相册与标签**：相册封面、公开 / 私有、分享链接
- **图片直出**：`/i/:hash` 原图、`/t/:hash/:size` 缩略图，支持动态裁剪与格式转换
- **回收站**：软删除、恢复、永久删除、定时清理
- **分享**：图片 / 相册分享，支持访问密码、有效期与访问次数上限
- **引用统计**：记录图片被哪些站点引用，提供 Top 域名与趋势图
- **API 令牌**：按权限范围（upload / read / write / delete / stats）签发令牌，明文仅显示一次
- **S3 兼容网关**：支持标准 S3 客户端使用 AccessKey / SecretKey 直传
- **管理后台**：仪表盘、用户、用户组、存储策略、系统设置、操作日志、邮件日志、邀请码
- **邮件**：邮箱验证、找回密码、SMTP 发送日志与失败重试
- **单二进制部署**：前端资源 `go:embed` 进二进制，一个进程即可运行

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.24、Gin、GORM、MySQL 8、Redis 7、Viper、zap |
| 前端 | Vue 3、TypeScript、Vite 5、Pinia、Vue Router 4、Element Plus、ECharts |
| 存储 | 本地磁盘、S3 兼容、阿里云 OSS、腾讯云 COS、WebDAV |
| 图片处理 | gen2brain/webp、动态裁剪 / 缩略图 / 水印 |

## 目录结构

```text
cmd/pichub            程序入口
configs               配置结构、内置模板、Docker 配置模板
migrations            SQL 迁移（启动时自动执行）
internal/
  bootstrap           依赖装配与生命周期
  config              配置加载、默认值、校验、密钥生成
  handler             HTTP 处理器（仅依赖 service 与 dto）
  service             业务逻辑与仓储接口（port.go）
  repository          数据访问实现（GORM）
  model               数据模型
  dto                 请求 / 响应结构
  middleware          登录态、限流、日志、恢复
  storage             存储驱动
  imageproc           缩略图、动态处理、水印
  queue               异步任务队列（Redis）
  mail                SMTP 与模板
  web                 go:embed 前端产物
web/                  Vue 3 前端工程
```

分层约束：`handler → service → repository → model`。handler 不直接依赖 GORM / repository，service 不依赖 Gin / net/http，仓储接口集中定义在 `internal/service/port.go`。

## 快速开始

### 方式一：Docker Compose（推荐）

```bash
# 1. 修改部署配置
#    - security.jwt_secret / secret_key / url_sign_secret
#    - database.password（与 compose 中的 MYSQL_PASSWORD 保持一致）
#    - site.url（对外访问地址）
vim configs/config.docker.yaml

# 2. 启动 MySQL、Redis 与 PicHub
docker compose up -d --build

# 3. 查看日志
docker compose logs -f pichub
```

访问 `http://localhost:8080`。**首个注册的用户会自动成为管理员**，可进入 `/admin` 管理后台。

> 默认配置关闭了邮箱验证（`register.require_email_verify: off`），方便本地试用。部署到公网前请配置 SMTP 并改为 `soft` 或 `force`。

### 方式二：本地源码运行

前置条件：Go 1.24+、Node.js 20+、MySQL 8、Redis 7。

```bash
# 安装依赖
make deps

# 生成配置（首次运行会自动填充随机密钥）
make config
vim config.yaml     # 修改 database / redis 连接信息

# 构建前端并编译单二进制
make build

# 运行
./bin/pichub -c config.yaml
```

开发模式（前后端分离）：

```bash
# 终端 1：后端
make run

# 终端 2：前端（Vite 开发服务器，代理 /api 到 8080）
make dev-web
```

若未提供配置文件，程序会按 `./config.yaml → /app/config.yaml → /etc/pichub/config.yaml` 查找；仍未找到时使用内置默认值并将模板写入 `./config.yaml`。密钥为空且 `security.auto_generate_secrets: true` 时会自动生成并回写配置文件。

## 配置说明

完整配置项见 [`configs/config.yaml`](configs/config.yaml)。

关键配置：

| 配置项 | 说明 |
| --- | --- |
| `site.url` | 对外访问地址，用于生成图片与分享链接 |
| `server.addr` | 监听地址，默认 `0.0.0.0:8080` |
| `database.*` / `redis.*` | 数据库与缓存连接 |
| `security.jwt_secret` | JWT 签名密钥，至少 32 字符 |
| `security.secret_key` | 敏感数据加密密钥（S3 Secret 等） |
| `security.url_sign_secret` | 私有图片签名 URL 密钥 |
| `register.mode` | `open` 开放注册 / `invite` 仅邀请码 |
| `register.require_email_verify` | `off` / `soft` / `force` |
| `upload.max_file_size` | 单文件大小上限（字节） |
| `upload.dedup_enabled` | 是否启用秒传 |
| `storage.default_driver` | `local` / `s3` / `oss` / `cos` / `webdav` |
| `mail.*` | SMTP 配置，启用邮箱验证时必填 |
| `s3_gateway.*` | S3 兼容网关开关、路径前缀与 Region |
| `log.*` | 日志级别、格式与输出位置 |

存储策略也可以在管理后台「存储策略」中动态创建，支持连通性测试，无需重启。

### 图片访问

- 原图：`GET /i/:hash`
- 缩略图：`GET /t/:hash/:size`（size 为配置中的 small / medium / large）
- 动态处理：`GET /i/:hash?w=800&h=600&fit=cover&q=85&format=webp`
    - `w` / `h`：目标宽高
    - `fit`：`cover` / `contain` / `fill`
    - `q`：质量 1-100
    - `format`：输出格式
    - `blur` / `gray`：模糊半径 / 灰度

私有图片需要签名参数 `uid`、`exp`、`sign`，由后端在接口响应中生成。

## API

所有接口前缀为 `/api/v1`，统一响应结构：

```json
{ "code": 0, "message": "ok", "data": {} }
```

- 登录后使用 `Authorization: Bearer <access_token>`
- 也可使用 API 令牌：`Authorization: Bearer ph_xxx`（在「API 令牌」页面创建）
- 业务错误使用对应 HTTP 状态码，`code` 为业务码

主要模块：

| 前缀 | 说明 |
| --- | --- |
| `/api/v1/auth` | 验证码、注册、登录、刷新、邮箱验证、找回密码 |
| `/api/v1/user` | 个人资料、修改密码、用量统计 |
| `/api/v1/images` | 图片列表、编辑、批量操作、回收站、引用来源 |
| `/api/v1/upload` | 上传、秒传检查、分片上传 |
| `/api/v1/albums` `/tags` | 相册与标签 |
| `/api/v1/tokens` `/s3/credentials` | API 令牌与 S3 凭据 |
| `/api/v1/shares` `/s/:slug` | 分享管理与公开访问 |
| `/api/v1/admin` | 管理后台接口 |

健康检查：`GET /healthz`、`GET /readyz`。

## S3 兼容网关

在「API 令牌 → S3 凭据」中创建 AccessKey / SecretKey，然后使用任意 S3 客户端：

```bash
export AWS_ACCESS_KEY_ID=PKIA1B2C3D4E5F6G7H8I
export AWS_SECRET_ACCESS_KEY=your-secret-key

# 上传
aws --endpoint-url http://localhost:8080/s3 s3 cp photo.jpg s3://pichub/photo.jpg
# 列举
aws --endpoint-url http://localhost:8080/s3 s3 ls s3://pichub/
```

- Endpoint：`<site.url>/s3`
- Region：`us-east-1`（与 `s3_gateway.region` 一致）
- Path Style：开启
- Bucket：`pichub`（对象 Key 即存储 Key，可通过 `<site.url>/s3/pichub/<key>` 访问）
- 支持单次 PUT、GET / HEAD / DELETE、ListObjectsV2 与 multipart 上传

## 部署

### 反向代理（Nginx 示例）

```nginx
server {
    listen 80;
    server_name img.example.com;
    client_max_body_size 50m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

部署后请同步修改 `site.url` 为 `https://img.example.com`，否则生成的图片链接不正确。

### 升级

```bash
docker compose pull    # 或重新 build
docker compose up -d
```

数据库迁移在启动时自动执行（`migrate.on_start`）。

## 安全建议

- 替换配置中的全部密钥与数据库密码，配置文件权限设为 `0600`
- 公网部署务必启用 HTTPS
- 配置 SMTP 并开启邮箱验证
- 按需关闭开放注册，改用邀请码
- 定期备份数据库与 `data/` 目录

## License

MIT
