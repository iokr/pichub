# PicHub 图床系统 · 完整设计文档

> **文档版本**：v1.0（最终版）
> **文档用途**：交付给 AI 作为实现依据，按本文档可直接完成开发
> **技术栈**：Go 1.25 + Gin + GORM + MySQL 8.0 + Redis 7 + Vue 3.4 + Element Plus + Docker
> **核心形态**：单二进制部署（前端嵌入后端），纯 YAML 配置，默认本地存储，开放多用户注册

---

## 目录

1. [项目概述](#1-项目概述)
2. [技术栈与版本](#2-技术栈与版本)
3. [系统架构](#3-系统架构)
4. [项目目录结构](#4-项目目录结构)
5. [数据库设计](#5-数据库设计)
6. [配置文件设计](#6-配置文件设计)
7. [后端模块设计](#7-后端模块设计)
8. [API 接口设计](#8-api-接口设计)
9. [前端设计](#9-前端设计)
10. [前后端一体化嵌入](#10-前后端一体化嵌入)
11. [Docker 部署](#11-docker-部署)
12. [关键特性实现](#12-关键特性实现)
13. [开发里程碑](#13-开发里程碑)
14. [验收清单](#14-验收清单)
15. [附录](#附录)

---

## 1. 项目概述

### 1.1 产品定位

PicHub 是一套可自托管的图床系统，面向个人博客作者、Markdown 写作者、小团队。核心价值：

- **开箱即用**：单二进制启动，默认本地存储，零外部依赖即可上传访问
- **多存储后端**：本地磁盘 / S3 兼容 / 阿里云 OSS / 腾讯云 COS / WebDAV
- **开放 API**：兼容 PicGo、Typora、Obsidian 等写作工具
- **S3 兼容网关**：直接对接 PicGo 的 S3 插件，上传入库不绕过系统
- **完整资产管理**：相册、标签、回收站、引用统计、分享页
- **三端响应式**：手机、平板、桌面一致体验

### 1.2 已确认的产品决策

| # | 决策项 | 结论 |
|---|---|---|
| 1 | 数据库 | MySQL 8.0 |
| 2 | 用户注册 | 开放注册（后台可关闭） |
| 3 | 图片默认可见性 | 公开 |
| 4 | 邮箱验证 | 默认 `force`（必须验证才能登录） |
| 5 | 人机校验 | 使用图片验证码（非 Turnstile） |
| 6 | CDN 自动刷新 | 不做 |
| 7 | 图片引用统计 | 需要 |
| 8 | 前端 UI 库 | Element Plus |
| 9 | 部署形态 | Docker 优先 |
| 10 | PicGo S3 插件直传 | 支持 |
| 11 | 响应式 | 手机 / 平板 / 桌面 |
| 12 | 数据库字段注释 | 全字段带 COMMENT |
| 13 | 前端视觉 | 美观、简洁 |
| 14 | 前端嵌入 | 编译进后端二进制，单服务访问 |
| 15 | 后端分层 | 严格的 handler / service / repository 分层 |
| 16 | 配置方式 | 纯 YAML，不使用环境变量 |
| 17 | 默认存储 | 本地磁盘 |

### 1.3 非目标（本期不做）

- 视频 / 音频 / 任意文件托管
- 多租户 SaaS 计费
- 图片 AI 打标 / 以图搜图
- 移动端原生 App
- CDN 自动刷新
- K8s Helm Chart（二期）

---

## 2. 技术栈与版本

### 2.1 后端

| 项 | 选型 | 版本 |
|---|---|---|
| 语言 | Go | 1.22+ |
| Web 框架 | Gin | v1.10.0 |
| ORM | GORM | v1.25.10 |
| MySQL 驱动 | gorm.io/driver/mysql | v1.5.7 |
| Redis 客户端 | go-redis | v9.5.3 |
| 配置 | Viper | v1.19.0 |
| 数据库迁移 | golang-migrate | v4.17.1 |
| JWT | golang-jwt/jwt | v5.2.1 |
| 日志 | zap | v1.27.0 |
| 定时任务 | robfig/cron | v3.0.1 |
| 图片验证码 | mojocn/base64Captcha | v1.3.6 |
| 图片处理 | disintegration/imaging | v1.6.2 |
| WebP 编码 | chai2010/webp | v1.1.1 |
| S3 SDK | aws-sdk-go-v2 | v1.30.0 |
| 参数校验 | go-playground/validator | v10.22.0 |
| API 文档 | swaggo/gin-swagger | v1.6.0 |

### 2.2 前端

| 项 | 选型 | 版本 |
|---|---|---|
| 框架 | Vue | 3.4+ |
| 语言 | TypeScript | 5.x |
| UI 库 | Element Plus | 2.7+ |
| 图标 | @element-plus/icons-vue | 2.3+ |
| 状态 | Pinia | 2.x |
| 路由 | Vue Router | 4.x |
| 请求 | Axios | 1.7+ |
| 构建 | Vite | 5.x |
| 样式 | SCSS + UnoCSS | - |
| 工具 | VueUse | 10.x |
| 虚拟滚动 | vue-virtual-scroller | 1.x |
| 图表 | ECharts + vue-echarts | 5.x / 7.x |

### 2.3 运行时依赖

| 项 | 版本 |
|---|---|
| MySQL | 8.0.30+ |
| Redis | 7.2+ |

---

## 3. 系统架构

### 3.1 部署形态

**单二进制**：前端静态资源通过 `go:embed` 编译进 Go 二进制，一个进程同时提供前端页面、API、图片直出、S3 兼容网关。

```
┌──────────────────────────────────────────────────────────┐
│                    pichub 单进程 (:8080)                  │
│                                                          │
│  ┌────────────────────────────────────────────────────┐  │
│  │  Gin Router                                        │  │
│  │  ┌──────────┬──────────┬──────────┬──────────────┐ │  │
│  │  │ /api/v1  │  /i /t   │  /s3     │  /assets /   │ │  │
│  │  │ 业务 API  │ 图片直出  │ S3 网关  │  前端 SPA    │ │  │
│  │  └──────────┴──────────┴──────────┴──────────────┘ │  │
│  └────────────────────────────────────────────────────┘  │
│                                                          │
│  ┌────────────┬────────────┬────────────┬────────────┐  │
│  │  Service   │ Repository │  Storage   │   Queue    │  │
│  │  业务逻辑   │  数据访问   │  存储抽象   │  异步任务   │  │
│  └────────────┴────────────┴────────────┴────────────┘  │
│                                                          │
│  ┌────────────┬────────────┬────────────────────────┐  │
│  │    Cron    │   Mailer   │   ImageProcessor       │  │
│  └────────────┴────────────┴────────────────────────┘  │
└───────────┬───────────────────────┬─────────────────────┘
            │                       │
     ┌──────▼──────┐         ┌──────▼──────┐
     │   MySQL     │         │    Redis    │
     │  元数据      │         │ 缓存/队列    │
     └─────────────┘         └─────────────┘
            │
     ┌──────▼──────────────────────────────┐
     │  存储后端（默认本地磁盘）              │
     │  local / s3 / oss / cos / webdav    │
     └─────────────────────────────────────┘
```

### 3.2 请求路由优先级

```
1. /healthz  /readyz  /metrics           探针
2. /api/v1/**                           业务 API
3. /swagger/**                          API 文档
4. /s3/**                               S3 兼容网关
5. /i/:hash  /t/:hash/:size             图片直出
6. /assets/**                           前端静态资源（长缓存）
7. /**（fallback）                       返回 index.html（SPA）
```

### 3.3 分层依赖方向（单向）

```
cmd → bootstrap → router → handler → service → repository → model
                          ↓         ↓
                     middleware   storage / imageproc / mail / queue
```

**硬性约束**：

- `handler` 不得直接 import `repository` 或 `gorm`
- `service` 不得 import `gin` / `net/http`
- `model` 不得 import 任何 internal 包
- 跨层调用通过接口，接口定义在**消费方**包（依赖倒置）

---

## 4. 项目目录结构

### 4.1 仓库根目录

```
pichub/
├── cmd/pichub/main.go               # 程序入口
├── internal/                        # 私有代码
├── pkg/                             # 可复用工具
├── migrations/                      # SQL 迁移文件
├── configs/                         # 配置模板
├── deploy/                          # 部署相关
├── web/                             # 前端工程
├── docs/                            # 文档
├── Dockerfile
├── Makefile
├── .gitignore
├── .golangci.yml
├── go.mod
├── go.sum
├── config.yaml.example
└── README.md
```

### 4.2 后端目录

```
internal/
├── bootstrap/                       # 依赖装配
│   ├── app.go                       # 组装所有组件，返回 *App
│   ├── db.go                        # MySQL 连接
│   ├── redis.go                     # Redis 连接
│   └── migrate.go                   # 迁移执行
│
├── config/                          # 配置加载
│   ├── config.go                    # Config 结构体
│   ├── defaults.go                  # 默认值
│   ├── load.go                      # 加载逻辑
│   ├── validate.go                  # 校验
│   └── secret.go                    # 密钥自动生成
│
├── router/                          # 路由注册
│   ├── router.go                    # 创建 Engine
│   ├── api.go                       # /api/v1
│   ├── admin.go                     # /api/v1/admin
│   ├── s3.go                        # /s3
│   ├── image.go                     # /i /t
│   ├── static.go                    # 前端 SPA
│   └── health.go                    # 探针
│
├── middleware/                      # 中间件
│   ├── auth.go                      # JWT 鉴权
│   ├── apitoken.go                  # API Token 鉴权
│   ├── s3auth.go                    # SigV4 校验
│   ├── admin.go                     # 管理员权限
│   ├── ratelimit.go                 # 限流
│   ├── cors.go
│   ├── recovery.go
│   ├── logger.go
│   └── requestid.go
│
├── handler/                         # HTTP 层（薄）
│   ├── auth_handler.go
│   ├── captcha_handler.go
│   ├── user_handler.go
│   ├── upload_handler.go
│   ├── image_handler.go
│   ├── album_handler.go
│   ├── tag_handler.go
│   ├── token_handler.go
│   ├── share_handler.go
│   ├── referer_handler.go
│   ├── storage_handler.go
│   ├── s3cred_handler.go
│   ├── stats_handler.go
│   ├── admin_handler.go
│   └── health_handler.go
│
├── service/                         # 业务逻辑层
│   ├── port.go                      # ⭐ 所有仓储接口定义
│   ├── auth_service.go
│   ├── user_service.go
│   ├── captcha_service.go
│   ├── mail_service.go
│   ├── upload_service.go
│   ├── image_service.go
│   ├── album_service.go
│   ├── tag_service.go
│   ├── token_service.go
│   ├── share_service.go
│   ├── referer_service.go
│   ├── storage_service.go
│   ├── s3gateway_service.go
│   ├── stats_service.go
│   ├── setting_service.go
│   └── invite_service.go
│
├── repository/                      # 数据访问层
│   ├── transaction.go               # 事务管理器
│   ├── user_repo.go
│   ├── user_group_repo.go
│   ├── image_repo.go
│   ├── album_repo.go
│   ├── tag_repo.go
│   ├── token_repo.go
│   ├── s3cred_repo.go
│   ├── share_repo.go
│   ├── referer_repo.go
│   ├── storage_repo.go
│   ├── email_repo.go
│   ├── setting_repo.go
│   ├── invite_repo.go
│   ├── log_repo.go
│   └── counter_repo.go
│
├── model/                           # 数据库实体
│   ├── base.go
│   ├── user.go
│   ├── user_group.go
│   ├── image.go
│   ├── album.go
│   ├── tag.go
│   ├── api_token.go
│   ├── s3_credential.go
│   ├── share.go
│   ├── image_referer.go
│   ├── storage_config.go
│   ├── email_token.go
│   ├── email_log.go
│   ├── setting.go
│   ├── invite.go
│   ├── operation_log.go
│   └── upload_counter.go
│
├── dto/                             # 请求/响应结构体
│   ├── common_dto.go                # 统一响应与分页
│   ├── auth_dto.go
│   ├── user_dto.go
│   ├── upload_dto.go
│   ├── image_dto.go
│   ├── album_dto.go
│   ├── tag_dto.go
│   ├── token_dto.go
│   ├── share_dto.go
│   ├── referer_dto.go
│   ├── storage_dto.go
│   ├── admin_dto.go
│   └── stats_dto.go
│
├── storage/                         # 存储驱动
│   ├── storage.go                   # Storage 接口
│   ├── factory.go                   # 驱动工厂
│   ├── manager.go                   # 策略管理器
│   ├── local/local.go
│   ├── s3/s3.go
│   ├── oss/oss.go
│   ├── cos/cos.go
│   └── webdav/webdav.go
│
├── imageproc/                       # 图片处理
│   ├── processor.go                 # 接口
│   ├── imaging.go                   # 纯 Go 实现
│   ├── thumbnail.go                 # 缩略图
│   ├── dynamic.go                   # 动态处理
│   ├── cache.go                     # LRU 磁盘缓存
│   └── options.go
│
├── s3gw/                            # S3 兼容网关
│   ├── sigv4.go                     # 签名校验
│   ├── handler.go                   # PutObject 等
│   ├── multipart.go                 # 分片
│   └── errors.go
│
├── cache/                           # Redis 封装
│   ├── client.go
│   ├── captcha.go
│   ├── ratelimit.go
│   ├── dedup.go
│   ├── referer_buffer.go
│   └── token_blacklist.go
│
├── queue/                           # 异步任务
│   ├── queue.go
│   ├── redis_queue.go
│   ├── worker.go
│   └── tasks/
│       ├── email_task.go
│       ├── thumbnail_task.go
│       ├── referer_flush_task.go
│       └── cleanup_task.go
│
├── cron/                            # 定时任务
│   ├── scheduler.go
│   ├── trash_cleanup.go
│   ├── referer_aggregate.go
│   ├── cache_cleanup.go
│   └── log_cleanup.go
│
├── mail/                            # 邮件
│   ├── sender.go
│   ├── smtp.go
│   └── templates/
│       ├── verify.html
│       ├── reset.html
│       └── change.html
│
├── auth/                            # 鉴权工具
│   ├── jwt.go
│   ├── password.go
│   ├── token.go
│   └── sign.go
│
├── crypto/                          # 加密
│   └── aes.go
│
├── errs/                            # 错误码
│   ├── code.go
│   ├── error.go
│   └── message.go
│
├── logger/                          # 日志
│   └── logger.go
│
└── web/                             # 前端嵌入
    ├── embed.go
    ├── noembed.go
    └── dist/                        # Vite 产物（.gitkeep 入仓）
```

### 4.3 前端目录

```
web/
├── index.html
├── package.json
├── vite.config.ts
├── tsconfig.json
├── uno.config.ts
├── .env.development
├── .env.production
└── src/
    ├── main.ts
    ├── App.vue
    ├── api/                         # 接口封装
    │   ├── client.ts                # axios 实例
    │   ├── auth.ts
    │   ├── upload.ts
    │   ├── image.ts
    │   ├── album.ts
    │   ├── tag.ts
    │   ├── token.ts
    │   ├── share.ts
    │   ├── referer.ts
    │   ├── storage.ts
    │   ├── stats.ts
    │   └── admin.ts
    ├── assets/
    │   └── images/
    ├── components/
    │   ├── common/
    │   │   ├── AppLogo.vue
    │   │   ├── AppTopBar.vue
    │   │   ├── AppSidebar.vue
    │   │   ├── AppTabBar.vue
    │   │   ├── EmptyState.vue
    │   │   ├── LoadingSkeleton.vue
    │   │   ├── PageHeader.vue
    │   │   ├── CaptchaField.vue
    │   │   ├── CopyButton.vue
    │   │   └── ThemeToggle.vue
    │   ├── upload/
    │   │   ├── UploadDropzone.vue
    │   │   ├── UploadQueue.vue
    │   │   ├── UploadCard.vue
    │   │   └── UploadParams.vue
    │   ├── image/
    │   │   ├── ImageGrid.vue
    │   │   ├── ImageCard.vue
    │   │   ├── ImageDetailDrawer.vue
    │   │   ├── ImageLinkList.vue
    │   │   └── RefererList.vue
    │   └── charts/
    │       ├── TrendChart.vue
    │       └── TopChart.vue
    ├── composables/
    │   ├── useDevice.ts
    │   ├── useTheme.ts
    │   ├── useCopy.ts
    │   └── useUpload.ts
    ├── layouts/
    │   ├── DefaultLayout.vue        # 桌面+平板
    │   ├── MobileLayout.vue
    │   └── BlankLayout.vue          # 登录/注册/分享
    ├── router/
    │   ├── index.ts
    │   └── guard.ts
    ├── stores/
    │   ├── user.ts
    │   ├── upload.ts
    │   ├── image.ts
    │   └── setting.ts
    ├── styles/
    │   ├── tokens/
    │   │   ├── _color.scss
    │   │   ├── _typography.scss
    │   │   └── _spacing.scss
    │   ├── element/index.scss
    │   ├── base.scss
    │   └── index.scss
    ├── types/
    │   └── api.d.ts                 # OpenAPI 生成
    ├── utils/
    │   ├── format.ts
    │   ├── hash.ts
    │   └── download.ts
    └── views/
        ├── auth/
        │   ├── LoginView.vue
        │   ├── RegisterView.vue
        │   ├── VerifyEmailView.vue
        │   └── ForgotPasswordView.vue
        ├── workspace/WorkspaceView.vue
        ├── gallery/GalleryView.vue
        ├── albums/
        │   ├── AlbumListView.vue
        │   └── AlbumDetailView.vue
        ├── trash/TrashView.vue
        ├── tokens/TokenView.vue
        ├── settings/SettingsView.vue
        ├── share/ShareView.vue
        ├── stats/StatsView.vue
        └── admin/
            ├── DashboardView.vue
            ├── UsersView.vue
            ├── GroupsView.vue
            ├── StoragesView.vue
            ├── SettingsView.vue
            ├── LogsView.vue
            └── InvitesView.vue
```

---

## 5. 数据库设计

### 5.1 全局约定

```sql
CREATE DATABASE IF NOT EXISTS `pichub`
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_0900_ai_ci;
USE `pichub`;
```

- 引擎 InnoDB，字符集 utf8mb4，排序 `utf8mb4_0900_ai_ci`
- 主键 `BIGINT UNSIGNED AUTO_INCREMENT`
- 时间 `DATETIME(3)`，统一 UTC
- 所有表、字段带 COMMENT
- 不使用物理外键，关联完整性由应用层保证
- 软删除用 `status` / `deleted_flag`，不用 GORM 内置 `DeletedAt`

### 5.2 完整 DDL

#### 5.2.1 用户组

```sql
CREATE TABLE `user_groups` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT           COMMENT '主键ID',
  `name`            VARCHAR(32)     NOT NULL                          COMMENT '用户组名称',
  `description`     VARCHAR(255)    NOT NULL DEFAULT ''               COMMENT '用户组描述',
  `quota_bytes`     BIGINT          NOT NULL DEFAULT 2147483648       COMMENT '总容量配额(字节)，0不限',
  `max_file_size`   BIGINT          NOT NULL DEFAULT 10485760         COMMENT '单文件大小上限(字节)',
  `daily_limit`     INT             NOT NULL DEFAULT 200              COMMENT '每日上传张数上限，0不限',
  `max_albums`      INT             NOT NULL DEFAULT 20               COMMENT '相册数量上限，0不限',
  `max_tokens`      INT             NOT NULL DEFAULT 5                COMMENT 'API Token数量上限，0不限',
  `max_shares`      INT             NOT NULL DEFAULT 50               COMMENT '分享链接数量上限，0不限',
  `allow_register`  TINYINT(1)      NOT NULL DEFAULT 1                COMMENT '是否允许自助注册进入该组',
  `allow_custom_storage` TINYINT(1) NOT NULL DEFAULT 0                COMMENT '是否允许自建存储策略',
  `is_default`      TINYINT(1)      NOT NULL DEFAULT 0                COMMENT '是否默认组',
  `status`          TINYINT         NOT NULL DEFAULT 1                COMMENT '状态：1启用 0禁用',
  `created_at`      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间(UTC)',
  `updated_at`      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间(UTC)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  KEY `idx_is_default` (`is_default`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户组：配额与权限策略';
```

#### 5.2.2 用户

```sql
CREATE TABLE `users` (
  `id`                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT      COMMENT '主键ID',
  `username`              VARCHAR(32)     NOT NULL                     COMMENT '登录用户名，大小写不敏感唯一',
  `email`                 VARCHAR(128)    NOT NULL                     COMMENT '邮箱地址，唯一',
  `password_hash`         VARCHAR(100)    NOT NULL                     COMMENT '密码哈希(bcrypt)',
  `nickname`              VARCHAR(32)     NOT NULL DEFAULT ''          COMMENT '昵称',
  `avatar_url`            VARCHAR(500)    NOT NULL DEFAULT ''          COMMENT '头像URL',
  `role`                  TINYINT         NOT NULL DEFAULT 0           COMMENT '角色：0用户 1管理员',
  `status`                TINYINT         NOT NULL DEFAULT 2           COMMENT '状态：1正常 0禁用 2待邮箱验证',
  `email_verified`        TINYINT(1)      NOT NULL DEFAULT 0           COMMENT '邮箱是否已验证',
  `group_id`              BIGINT UNSIGNED NOT NULL DEFAULT 1           COMMENT '用户组ID',
  `quota_bytes`           BIGINT          NOT NULL DEFAULT 0           COMMENT '独立容量配额，0继承用户组',
  `used_bytes`            BIGINT          NOT NULL DEFAULT 0           COMMENT '已使用容量(字节)',
  `max_file_size`         BIGINT          NOT NULL DEFAULT 0           COMMENT '单文件上限，0继承用户组',
  `daily_limit`           INT             NOT NULL DEFAULT 0           COMMENT '每日上传上限，0继承用户组',
  `default_visibility`    VARCHAR(10)     NOT NULL DEFAULT 'public'    COMMENT '默认可见性：public/private',
  `storage_config_id`     BIGINT UNSIGNED NULL                         COMMENT '默认存储策略ID，NULL用系统默认',
  `referer_stat_enabled`  TINYINT(1)      NOT NULL DEFAULT 1           COMMENT '是否开启引用统计',
  `register_ip`           VARBINARY(16)   NOT NULL DEFAULT ''          COMMENT '注册IP',
  `last_login_ip`         VARBINARY(16)   NOT NULL DEFAULT ''          COMMENT '最后登录IP',
  `last_login_at`         DATETIME(3)     NULL                         COMMENT '最后登录时间',
  `created_at`            DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '注册时间(UTC)',
  `updated_at`            DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间(UTC)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  UNIQUE KEY `uk_email` (`email`),
  KEY `idx_status_created` (`status`, `created_at`),
  KEY `idx_group` (`group_id`),
  KEY `idx_register_ip` (`register_ip`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户账号';
```

#### 5.2.3 邮箱令牌与日志

```sql
CREATE TABLE `email_tokens` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT               COMMENT '主键ID',
  `user_id`     BIGINT UNSIGNED NOT NULL                              COMMENT '用户ID',
  `email`       VARCHAR(128)    NOT NULL                              COMMENT '目标邮箱',
  `token_hash`  CHAR(64)        NOT NULL                              COMMENT '令牌哈希(sha256)',
  `scene`       VARCHAR(20)     NOT NULL                              COMMENT '场景：verify/reset/change',
  `expires_at`  DATETIME(3)     NOT NULL                              COMMENT '过期时间',
  `used_at`     DATETIME(3)     NULL                                  COMMENT '使用时间',
  `created_at`  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_token_hash` (`token_hash`),
  KEY `idx_user_scene` (`user_id`, `scene`, `created_at` DESC),
  KEY `idx_expires` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='邮箱令牌';

CREATE TABLE `email_logs` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT               COMMENT '主键ID',
  `user_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0                    COMMENT '用户ID，0为系统邮件',
  `to_email`    VARCHAR(128)    NOT NULL                              COMMENT '收件人邮箱',
  `subject`     VARCHAR(255)    NOT NULL DEFAULT ''                   COMMENT '邮件主题',
  `template`    VARCHAR(32)     NOT NULL                              COMMENT '模板标识',
  `status`      TINYINT         NOT NULL DEFAULT 0                    COMMENT '状态：0待发 1成功 2失败',
  `error`       VARCHAR(500)    NOT NULL DEFAULT ''                   COMMENT '失败原因',
  `retry`       TINYINT         NOT NULL DEFAULT 0                    COMMENT '已重试次数',
  `sent_at`     DATETIME(3)     NULL                                  COMMENT '发送成功时间',
  `created_at`  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_status_created` (`status`, `created_at`),
  KEY `idx_user` (`user_id`, `created_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='邮件发送日志';
```

#### 5.2.4 API Token

```sql
CREATE TABLE `api_tokens` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT            COMMENT '主键ID',
  `user_id`        BIGINT UNSIGNED NOT NULL                           COMMENT '用户ID',
  `name`           VARCHAR(64)     NOT NULL                           COMMENT '令牌名称',
  `token_hash`     CHAR(64)        NOT NULL                           COMMENT '令牌哈希(sha256)',
  `token_prefix`   VARCHAR(12)     NOT NULL                           COMMENT '令牌前缀(展示用)',
  `scopes`         VARCHAR(255)    NOT NULL DEFAULT 'upload,read'     COMMENT '权限范围',
  `expires_at`     DATETIME(3)     NULL                               COMMENT '过期时间，NULL永久',
  `last_used_at`   DATETIME(3)     NULL                               COMMENT '最后使用时间',
  `last_used_ip`   VARBINARY(16)   NOT NULL DEFAULT ''                COMMENT '最后使用IP',
  `call_count`     BIGINT          NOT NULL DEFAULT 0                 COMMENT '累计调用次数',
  `status`         TINYINT         NOT NULL DEFAULT 1                 COMMENT '状态：1启用 0禁用',
  `created_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_token_hash` (`token_hash`),
  KEY `idx_user` (`user_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='API访问令牌';
```

#### 5.2.5 S3 兼容凭据

```sql
CREATE TABLE `s3_credentials` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT             COMMENT '主键ID',
  `user_id`       BIGINT UNSIGNED NOT NULL                            COMMENT '用户ID',
  `access_key`    VARCHAR(32)     NOT NULL                            COMMENT 'Access Key ID',
  `secret_enc`    VARBINARY(256)  NOT NULL                            COMMENT 'Secret Key(AES-GCM加密)',
  `name`          VARCHAR(64)     NOT NULL DEFAULT 'default'          COMMENT '凭据名称',
  `status`        TINYINT         NOT NULL DEFAULT 1                  COMMENT '状态：1启用 0禁用',
  `last_used_at`  DATETIME(3)     NULL                                COMMENT '最后使用时间',
  `last_used_ip`  VARBINARY(16)   NOT NULL DEFAULT ''                 COMMENT '最后使用IP',
  `call_count`    BIGINT          NOT NULL DEFAULT 0                  COMMENT '累计调用次数',
  `created_at`    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_access_key` (`access_key`),
  KEY `idx_user` (`user_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='S3兼容凭据';
```

#### 5.2.6 存储策略

```sql
CREATE TABLE `storage_configs` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT             COMMENT '主键ID',
  `user_id`       BIGINT UNSIGNED NULL                                COMMENT '用户ID，NULL为系统级',
  `name`          VARCHAR(64)     NOT NULL                            COMMENT '策略名称',
  `driver`        VARCHAR(20)     NOT NULL                            COMMENT '驱动：local/s3/oss/cos/webdav',
  `config`        JSON            NOT NULL                            COMMENT '非敏感配置',
  `secret`        VARBINARY(1024) NOT NULL                            COMMENT '敏感配置(AES-GCM)',
  `domain`        VARCHAR(255)    NOT NULL DEFAULT ''                 COMMENT '访问域名',
  `path_prefix`   VARCHAR(128)    NOT NULL DEFAULT 'uploads'          COMMENT '路径前缀',
  `is_default`    TINYINT(1)      NOT NULL DEFAULT 0                  COMMENT '是否默认策略',
  `status`        TINYINT         NOT NULL DEFAULT 1                  COMMENT '状态：1启用 0禁用',
  `last_test_at`  DATETIME(3)     NULL                                COMMENT '最后测试时间',
  `last_test_ok`  TINYINT(1)      NULL                                COMMENT '最后测试结果',
  `created_at`    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at`    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`, `status`),
  KEY `idx_default` (`is_default`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='存储策略配置';
```

#### 5.2.7 相册与图片

```sql
CREATE TABLE `albums` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT            COMMENT '主键ID',
  `user_id`        BIGINT UNSIGNED NOT NULL                           COMMENT '用户ID',
  `name`           VARCHAR(64)     NOT NULL                           COMMENT '相册名称',
  `description`    VARCHAR(255)    NOT NULL DEFAULT ''                COMMENT '相册描述',
  `cover_image_id` BIGINT UNSIGNED NULL                               COMMENT '封面图片ID',
  `is_public`      TINYINT(1)      NOT NULL DEFAULT 0                 COMMENT '是否公开',
  `sort_order`     INT             NOT NULL DEFAULT 0                 COMMENT '排序序号',
  `image_count`    INT             NOT NULL DEFAULT 0                 COMMENT '图片数(冗余)',
  `created_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_sort` (`user_id`, `sort_order`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='相册';

CREATE TABLE `images` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT            COMMENT '主键ID',
  `user_id`        BIGINT UNSIGNED NOT NULL                           COMMENT '用户ID',
  `album_id`       BIGINT UNSIGNED NULL                               COMMENT '相册ID，NULL未分组',
  `hash`           CHAR(64)        NOT NULL                           COMMENT '文件SHA-256',
  `origin_name`    VARCHAR(255)    NOT NULL DEFAULT ''                COMMENT '原始文件名',
  `display_name`   VARCHAR(255)    NOT NULL DEFAULT ''                COMMENT '显示名称',
  `ext`            VARCHAR(10)     NOT NULL                           COMMENT '扩展名',
  `mime`           VARCHAR(64)     NOT NULL                           COMMENT 'MIME类型',
  `size`           BIGINT          NOT NULL                           COMMENT '文件大小(字节)',
  `width`          INT             NOT NULL DEFAULT 0                 COMMENT '宽度(像素)',
  `height`         INT             NOT NULL DEFAULT 0                 COMMENT '高度(像素)',
  `storage_driver` VARCHAR(20)     NOT NULL                           COMMENT '存储驱动',
  `storage_key`    VARCHAR(500)    NOT NULL                           COMMENT '存储Key',
  `url`            VARCHAR(768)    NOT NULL                           COMMENT '主访问URL',
  `thumb_key`      VARCHAR(500)    NOT NULL DEFAULT ''                COMMENT '缩略图Key',
  `is_public`      TINYINT(1)      NOT NULL DEFAULT 1                 COMMENT '是否公开',
  `ref_count`      INT             NOT NULL DEFAULT 1                 COMMENT '物理文件引用计数',
  `status`         TINYINT         NOT NULL DEFAULT 1                 COMMENT '状态：1正常 0回收站',
  `deleted_flag`   BIGINT          NOT NULL DEFAULT 0                 COMMENT '删除标记：0未删，删除写时间戳',
  `deleted_at`     DATETIME(3)     NULL                               COMMENT '删除时间',
  `created_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '上传时间',
  `updated_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_hash_alive` (`user_id`, `hash`, `deleted_flag`),
  KEY `idx_user_created` (`user_id`, `status`, `created_at` DESC, `id` DESC),
  KEY `idx_user_album` (`user_id`, `album_id`, `created_at` DESC),
  KEY `idx_hash` (`hash`),
  KEY `idx_status_del` (`status`, `deleted_at`),
  KEY `idx_ext` (`ext`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='图片元数据';
```

#### 5.2.8 标签

```sql
CREATE TABLE `tags` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT               COMMENT '主键ID',
  `user_id`     BIGINT UNSIGNED NOT NULL                              COMMENT '用户ID',
  `name`        VARCHAR(32)     NOT NULL                              COMMENT '标签名称',
  `color`       VARCHAR(16)     NOT NULL DEFAULT ''                   COMMENT '标签颜色',
  `image_count` INT             NOT NULL DEFAULT 0                    COMMENT '关联图片数(冗余)',
  `created_at`  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_name` (`user_id`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='标签';

CREATE TABLE `image_tags` (
  `image_id`   BIGINT UNSIGNED NOT NULL                               COMMENT '图片ID',
  `tag_id`     BIGINT UNSIGNED NOT NULL                               COMMENT '标签ID',
  `created_at` DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3)  COMMENT '创建时间',
  PRIMARY KEY (`image_id`, `tag_id`),
  KEY `idx_tag` (`tag_id`, `image_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='图片-标签关联';
```

#### 5.2.9 分享与引用

```sql
CREATE TABLE `shares` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT            COMMENT '主键ID',
  `user_id`        BIGINT UNSIGNED NOT NULL                           COMMENT '创建者ID',
  `slug`           VARCHAR(16)     NOT NULL                           COMMENT '短链标识',
  `target_type`    VARCHAR(10)     NOT NULL                           COMMENT '类型：image/album',
  `target_id`      BIGINT UNSIGNED NOT NULL                           COMMENT '目标ID',
  `password_hash`  VARCHAR(100)    NOT NULL DEFAULT ''                COMMENT '密码哈希',
  `expires_at`     DATETIME(3)     NULL                               COMMENT '过期时间',
  `max_views`      INT             NOT NULL DEFAULT 0                 COMMENT '最大访问次数，0不限',
  `views`          INT             NOT NULL DEFAULT 0                 COMMENT '已访问次数',
  `status`         TINYINT         NOT NULL DEFAULT 1                 COMMENT '状态：1有效 0失效',
  `created_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_slug` (`slug`),
  KEY `idx_user_created` (`user_id`, `created_at` DESC),
  KEY `idx_target` (`target_type`, `target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='分享链接';

CREATE TABLE `image_referers` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT            COMMENT '主键ID',
  `image_id`       BIGINT UNSIGNED NOT NULL                           COMMENT '图片ID',
  `referer_hash`   CHAR(64)        NOT NULL                           COMMENT 'Referer URL的sha256',
  `referer_url`    VARCHAR(1024)   NOT NULL DEFAULT ''                COMMENT '完整Referer URL',
  `referer_host`   VARCHAR(255)    NOT NULL DEFAULT ''                COMMENT 'Referer域名',
  `hit_count`      BIGINT          NOT NULL DEFAULT 0                 COMMENT '累计次数',
  `first_seen_at`  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '首次发现时间',
  `last_seen_at`   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '最近发现时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_img_ref` (`image_id`, `referer_hash`),
  KEY `idx_image_hits` (`image_id`, `hit_count` DESC),
  KEY `idx_host` (`referer_host`, `last_seen_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='图片引用来源';

CREATE TABLE `image_referer_daily` (
  `image_id`     BIGINT UNSIGNED NOT NULL                             COMMENT '图片ID',
  `stat_date`    DATE            NOT NULL                             COMMENT '统计日期',
  `referer_host` VARCHAR(255)    NOT NULL                             COMMENT 'Referer域名',
  `hit_count`    BIGINT          NOT NULL DEFAULT 0                   COMMENT '当日次数',
  PRIMARY KEY (`image_id`, `stat_date`, `referer_host`),
  KEY `idx_date` (`stat_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='图片引用按天聚合';
```

#### 5.2.10 用量、日志、设置、邀请

```sql
CREATE TABLE `upload_counters` (
  `user_id`    BIGINT UNSIGNED NOT NULL                               COMMENT '用户ID',
  `stat_date`  DATE            NOT NULL                               COMMENT '统计日期(UTC)',
  `count`      INT             NOT NULL DEFAULT 0                     COMMENT '当日上传张数',
  `bytes`      BIGINT          NOT NULL DEFAULT 0                     COMMENT '当日上传字节数',
  `updated_at` DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`user_id`, `stat_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='每日上传用量';

CREATE TABLE `operation_logs` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT               COMMENT '主键ID',
  `user_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0                    COMMENT '操作者ID',
  `action`      VARCHAR(48)     NOT NULL                              COMMENT '操作类型',
  `target_type` VARCHAR(24)     NOT NULL DEFAULT ''                   COMMENT '对象类型',
  `target_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0                    COMMENT '对象ID',
  `ip`          VARBINARY(16)   NOT NULL DEFAULT ''                   COMMENT '来源IP',
  `user_agent`  VARCHAR(255)    NOT NULL DEFAULT ''                   COMMENT 'User-Agent',
  `detail`      JSON            NULL                                  COMMENT '操作详情',
  `created_at`  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '操作时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_created` (`user_id`, `created_at` DESC),
  KEY `idx_action_created` (`action`, `created_at` DESC),
  KEY `idx_target` (`target_type`, `target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='操作审计日志';

CREATE TABLE `system_settings` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT               COMMENT '主键ID',
  `cfg_key`     VARCHAR(64)     NOT NULL                              COMMENT '配置键',
  `cfg_value`   TEXT            NOT NULL                              COMMENT '配置值',
  `value_type`  VARCHAR(16)     NOT NULL DEFAULT 'string'             COMMENT '值类型',
  `is_secret`   TINYINT(1)      NOT NULL DEFAULT 0                    COMMENT '是否加密存储',
  `description` VARCHAR(255)    NOT NULL DEFAULT ''                   COMMENT '配置说明',
  `updated_at`  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cfg_key` (`cfg_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='系统全局配置';

CREATE TABLE `invites` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT               COMMENT '主键ID',
  `code`        VARCHAR(32)     NOT NULL                              COMMENT '邀请码',
  `creator_id`  BIGINT UNSIGNED NOT NULL                              COMMENT '创建者ID',
  `max_uses`    INT             NOT NULL DEFAULT 1                    COMMENT '最大使用次数，0不限',
  `used_count`  INT             NOT NULL DEFAULT 0                    COMMENT '已使用次数',
  `group_id`    BIGINT UNSIGNED NOT NULL DEFAULT 1                    COMMENT '注册后加入的用户组ID',
  `expires_at`  DATETIME(3)     NULL                                  COMMENT '过期时间',
  `status`      TINYINT         NOT NULL DEFAULT 1                    COMMENT '状态：1有效 0失效',
  `created_at`  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`),
  KEY `idx_creator` (`creator_id`, `created_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='注册邀请码';
```

### 5.3 初始化数据

```sql
INSERT INTO `user_groups`
  (`name`, `description`, `quota_bytes`, `max_file_size`, `daily_limit`, `is_default`, `allow_register`)
VALUES
  ('默认组',   '新注册用户默认加入', 2147483648,  10485760, 200,  1, 1),
  ('信任用户', '管理员手动升级',     21474836480, 20971520, 2000, 0, 0),
  ('VIP',      '高级用户',          0,           52428800, 0,    0, 0),
  ('受限',     '违规降级用户',       209715200,   5242880,  20,   0, 0);

INSERT INTO `system_settings` (`cfg_key`, `cfg_value`, `value_type`, `description`) VALUES
  ('site.name',                     'PicHub',  'string', '站点名称'),
  ('site.url',                      '',        'string', '站点对外地址'),
  ('register.enabled',              'true',    'bool',   '是否开放注册'),
  ('register.mode',                 'open',    'string', '注册模式：open/invite'),
  ('register.require_email_verify', 'force',   'string', '邮箱验证级别：off/soft/force'),
  ('register.ip_limit_per_day',     '3',       'int',    '单IP每日注册上限'),
  ('register.default_group_id',     '1',       'int',    '默认用户组ID'),
  ('captcha.enabled',               'true',    'bool',   '是否启用图片验证码'),
  ('captcha.type',                  'string',  'string', '验证码类型'),
  ('captcha.length',                '4',       'int',    '验证码长度'),
  ('captcha.expire_seconds',        '300',     'int',    '验证码有效期(秒)'),
  ('captcha.login_trigger_fails',   '2',       'int',    '登录失败几次触发验证码'),
  ('upload.default_visibility',     'public',  'string', '默认上传可见性'),
  ('upload.allowed_extensions',     '["jpg","jpeg","png","gif","webp","bmp","svg","avif","ico"]', 'json', '允许的扩展名'),
  ('referer.enabled',               'true',    'bool',   '是否启用引用统计'),
  ('referer.max_per_image',         '200',     'int',    '单图引用记录上限'),
  ('s3_gateway.enabled',            'true',    'bool',   '是否启用S3兼容网关'),
  ('s3_gateway.path_prefix',        '/s3',     'string', 'S3网关路径前缀');
```

---

## 6. 配置文件设计

### 6.1 设计原则

- **纯 YAML**，不使用环境变量
- 加载优先级：`-c 参数` > `./config.yaml` > `/app/config.yaml` > `/etc/pichub/config.yaml` > 内置默认值
- 严格模式：未知字段报错（防止拼写错误静默失效）
- 首次启动自动生成 `jwt_secret` 与 `secret_key` 并回写 YAML
- 启动日志打印实际加载的配置文件绝对路径
- 非 0600 权限时打印警告

### 6.2 完整配置模板

```yaml
# ============================================================================
# PicHub 图床系统配置文件
# 安全提示：本文件包含敏感信息，请设置权限为 0600
#   chmod 600 config.yaml && chown pichub:pichub config.yaml
# ============================================================================

# ---------------------------- 站点 ----------------------------
site:
  name: "PicHub"
  url: "http://localhost:8080"
  base_path: "/"
  description: "简洁优雅的图床系统"
  logo_url: ""
  icp: ""

# ---------------------------- 服务 ----------------------------
server:
  addr: "0.0.0.0:8080"
  mode: "release"                       # debug / release / test
  read_timeout: 60s
  write_timeout: 120s
  idle_timeout: 120s
  max_header_bytes: 1048576
  trusted_proxies:
    - "127.0.0.1"
    - "10.0.0.0/8"
    - "172.16.0.0/12"
    - "192.168.0.0/16"

# ---------------------------- 数据库 ----------------------------
database:
  host: "127.0.0.1"
  port: 3306
  username: "pichub"
  password: "CHANGE_ME_DB_PASSWORD"
  database: "pichub"
  charset: "utf8mb4"
  collation: "utf8mb4_0900_ai_ci"
  loc: "UTC"
  parse_time: true
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 30m
  conn_max_idle_time: 10m
  slow_threshold: 200ms
  log_level: "warn"                     # silent / error / warn / info

# ---------------------------- Redis ----------------------------
redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0
  pool_size: 50
  min_idle_conns: 10
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
  key_prefix: "pichub:"

# ---------------------------- 安全 ----------------------------
security:
  jwt_secret: "CHANGE_ME_JWT_SECRET_AT_LEAST_32_CHARS"
  secret_key: "CHANGE_ME_SECRET_KEY_BASE64_32_BYTES"
  jwt_access_ttl: "2h"
  jwt_refresh_ttl: "168h"
  url_sign_secret: ""
  bcrypt_cost: 10
  password_min_length: 8
  password_require_letter: true
  password_require_digit: true
  password_require_special: false
  login_max_fails: 5
  login_lock_duration: 10m
  auto_generate_secrets: true           # 密钥为空时自动生成并回写
  rate_limit:
    enabled: true
    login_per_minute: 10
    register_per_hour: 10
    upload_per_minute: 60
    api_per_minute: 300
    image_per_minute: 600

# ---------------------------- 注册 ----------------------------
register:
  enabled: true
  mode: "open"                          # open / invite
  require_email_verify: "force"         # off / soft / force
  default_group_id: 1
  ip_limit_per_day: 3
  username_min_length: 3
  username_max_length: 32
  username_pattern: "^[a-zA-Z0-9_-]+$"
  reserved_usernames: [admin, root, system, api, www, test, support]
  email_domain_blocklist:
    - "tempmail.com"
    - "10minutemail.com"
    - "guerrillamail.com"

# ---------------------------- 验证码 ----------------------------
captcha:
  enabled: true
  type: "string"                        # string / math / chinese
  length: 4
  width: 240
  height: 80
  noise_count: 20
  line_count: 3
  expire_seconds: 300
  login_trigger_fails: 2
  store_driver: "redis"                 # redis / memory

# ---------------------------- 上传 ----------------------------
upload:
  default_visibility: "public"
  max_file_size: 10485760
  allowed_extensions: [jpg, jpeg, png, gif, webp, bmp, svg, avif, ico]
  allow_svg: false
  chunk_size: 5242880
  max_chunk_count: 1000
  dedup_enabled: true
  dedup_scope: "user"                   # user / global
  keep_origin_name: true
  concurrent_limit: 3

# ---------------------------- 存储 ----------------------------
storage:
  # 默认使用本地存储
  default_driver: "local"
  default_strategy_name: "本地存储"
  allow_user_strategy: false

  local:
    root_path: "./data/uploads"
    path_prefix: "uploads"
    url_prefix: "/i"
    thumb_path: "./data/thumbs"
    dir_permission: 0755
    file_permission: 0644
    path_template: "{prefix}/{yyyy}/{MM}/{dd}/{hash2}/{hash}.{ext}"

  # 以下驱动默认不启用，需要时切换 default_driver
  s3:
    endpoint: ""
    region: "us-east-1"
    bucket: ""
    access_key: ""
    secret_key: ""
    path_style: false
    use_ssl: true
    public_url: ""

  oss:
    endpoint: ""
    bucket: ""
    access_key: ""
    secret_key: ""
    public_url: ""

  cos:
    endpoint: ""
    bucket: ""
    access_key: ""
    secret_key: ""
    public_url: ""

  webdav:
    url: ""
    username: ""
    password: ""
    public_url: ""

# ---------------------------- 图片处理 ----------------------------
image:
  thumbnail:
    enabled: true
    sizes:
      - {name: "small",  width: 200}
      - {name: "medium", width: 800}
      - {name: "large",  width: 1600}
    format: "webp"
    quality: 82

  dynamic:
    enabled: true
    max_width: 4096
    max_height: 4096
    default_quality: 85
    allowed_formats: [webp, jpeg, png, avif]
    cache_enabled: true
    cache_dir: "./data/cache/imageproc"
    cache_max_size: 10737418240
    cache_ttl: "168h"

  watermark:
    enabled: false
    type: "text"
    text: ""
    image_path: ""
    position: "bottom-right"
    opacity: 0.5
    font_size: 24
    color: "#FFFFFF"

  anti_leech:
    enabled: false
    allow_empty_referer: true
    referer_whitelist: []
    block_action: "redirect"            # redirect / forbidden

# ---------------------------- 引用统计 ----------------------------
referer:
  enabled: true
  max_per_image: 200
  buffer_size: 10000
  flush_interval: 60s
  daily_retain_days: 180
  user_can_disable: true
  exclude_bots: true
  bot_user_agents: [Googlebot, bingbot, Baiduspider, YandexBot, Bytespider, DuckDuckBot]

# ---------------------------- 邮件 ----------------------------
mail:
  enabled: false                        # 启用邮箱验证时需设为 true
  host: "smtp.example.com"
  port: 465
  username: "noreply@example.com"
  password: ""
  from_name: "PicHub"
  from_email: "noreply@example.com"
  encryption: "ssl"                     # ssl / starttls / none
  daily_limit: 500
  timeout: 10s
  retry_times: 3

# ---------------------------- S3 兼容网关 ----------------------------
s3_gateway:
  enabled: true
  path_prefix: "/s3"
  region: "us-east-1"
  clock_skew: 15m
  replay_window: 15m
  allow_multipart: true
  part_size: 5242880

# ---------------------------- 定时任务 ----------------------------
cron:
  enabled: true
  trash_cleanup:
    enabled: true
    schedule: "0 3 * * *"
    retain_days: 30
  referer_aggregate:
    enabled: true
    schedule: "0 4 * * *"
  cache_cleanup:
    enabled: true
    schedule: "0 5 * * *"
  email_log_cleanup:
    enabled: true
    schedule: "0 6 * * *"
    retain_days: 30
  operation_log_cleanup:
    enabled: true
    schedule: "0 6 * * 0"
    retain_days: 90

# ---------------------------- 队列 ----------------------------
queue:
  worker_count: 4
  max_retry: 3
  retry_delay: 30s
  task_timeout: 300s

# ---------------------------- 迁移 ----------------------------
migrate:
  on_start: true

# ---------------------------- 日志 ----------------------------
log:
  level: "info"
  format: "json"                        # json / console
  output: "both"                        # stdout / file / both
  file_path: "./data/logs/pichub.log"
  max_size: 100
  max_backups: 30
  max_age: 30
  compress: true
  color: false
  caller: true

# ---------------------------- 监控 ----------------------------
monitor:
  pprof_enabled: false
  pprof_addr: "127.0.0.1:6060"
  metrics_enabled: false
  metrics_path: "/metrics"
```

---

## 7. 后端模块设计

### 7.1 分层职责

| 层 | 职责 | 禁止事项 |
|---|---|---|
| `cmd` | 程序入口，解析 flag，装配，优雅关闭 | 不写业务逻辑 |
| `bootstrap` | 依赖装配（DB → Redis → Repo → Service → Handler → Router） | 不写业务逻辑 |
| `router` | 路由注册、分组、中间件挂载 | 不写业务逻辑 |
| `middleware` | 横切关注点 | 不写业务逻辑 |
| `handler` | 参数绑定、校验、调 service、组装响应 | 不碰 DB，不 import repository/gorm |
| `service` | 业务规则、事务边界、多 repo 编排 | 不 import gin/net/http |
| `repository` | 数据访问，只操作 MySQL | 不 import service，只返回 model |
| `model` | GORM 实体 | 不 import 任何 internal 包 |
| `dto` | 请求/响应结构体 | - |

### 7.2 依赖倒置（接口在消费方定义）

**`internal/service/port.go`**

```go
package service

import (
	"context"
	"github.com/pichub/pichub/internal/model"
)

type ImageRepo interface {
	Create(ctx context.Context, img *model.Image) error
	GetByID(ctx context.Context, id uint64) (*model.Image, error)
	GetByUserHash(ctx context.Context, userID uint64, hash string) (*model.Image, error)
	List(ctx context.Context, q ImageQuery) ([]*model.Image, int64, error)
	SoftDelete(ctx context.Context, id uint64) error
	Restore(ctx context.Context, id uint64) error
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
	IncrRefCount(ctx context.Context, hash string, delta int) error
}

type UserRepo interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id uint64) (*model.User, error)
	GetByUsername(ctx context.Context, name string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Count(ctx context.Context) (int64, error)
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
	AddUsedBytes(ctx context.Context, id uint64, delta int64) (bool, error)
}

type Transaction interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// 其余 Repo 接口按此模式定义
```

### 7.3 关键实现骨架

#### `cmd/pichub/main.go`

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pichub/pichub/internal/bootstrap"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/logger"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	cfgPath := flag.String("c", "", "config file path")
	showVer := flag.Bool("v", false, "show version")
	flag.Parse()

	if *showVer {
		fmt.Printf("pichub %s (built %s)\n", version, buildTime)
		return
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(&cfg.Log)
	log.Info("starting pichub",
		logger.String("version", version),
		logger.String("config", cfg.LoadedPath()))

	app, err := bootstrap.New(cfg, log)
	if err != nil {
		log.Fatal("bootstrap failed", logger.Err(err))
	}

	go func() {
		if err := app.Run(); err != nil {
			log.Fatal("server exited", logger.Err(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = app.Shutdown(ctx)
	log.Info("bye")
}
```

#### `internal/bootstrap/app.go`

```go
package bootstrap

type App struct {
	cfg     *config.Config
	log     *logger.Logger
	httpSrv *http.Server
	worker  *queue.Worker
	cron    *cron.Scheduler
}

func New(cfg *config.Config, log *logger.Logger) (*App, error) {
	db, err := NewDB(cfg, log)
	if err != nil { return nil, err }
	if cfg.Migrate.OnStart {
		if err := RunMigrate(cfg, log); err != nil { return nil, err }
	}
	rdb, err := NewRedis(cfg, log)
	if err != nil { return nil, err }

	repos := repository.New(db)
	storages := storage.NewManager(cfg, repos.Storage, log)
	processor := imageproc.New(cfg.Image, log)
	mailer := mail.New(cfg.Mail, log)
	q := queue.NewRedisQueue(rdb, log)
	captcha := cache.NewCaptcha(rdb, cfg.Captcha)

	services := service.New(service.Deps{
		Cfg: cfg, Log: log, Repos: repos,
		Storages: storages, Processor: processor,
		Mailer: mailer, Queue: q, Captcha: captcha,
	})

	engine := router.New(cfg, log, services, repos)

	return &App{
		cfg:     cfg,
		log:     log,
		httpSrv: &http.Server{Addr: cfg.Server.Addr, Handler: engine},
		worker:  queue.NewWorker(q, services, log),
		cron:    cron.New(&cfg.Cron, services, log),
	}, nil
}
```

#### `internal/handler/image_handler.go`（薄）

```go
package handler

func (h *ImageHandler) List(c *gin.Context) {
	var req dto.ImageListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		dto.Fail(c, errs.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	req.UserID = GetUserID(c)

	res, err := h.svc.List(c.Request.Context(), &req)
	if err != nil {
		dto.Fail(c, err)
		return
	}
	dto.OK(c, res)
}
```

#### `internal/service/image_service.go`（业务，无 gin）

```go
func (s *ImageService) SoftDelete(ctx context.Context, userID, imageID uint64) error {
	img, err := s.images.GetByID(ctx, imageID)
	if err != nil { return errs.ErrInternal.Wrap(err) }
	if img == nil || img.UserID != userID {
		return errs.ErrNotFound
	}
	return s.images.SoftDelete(ctx, imageID)
}
```

#### `internal/dto/common_dto.go`

```go
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "ok", Data: data})
}

func Fail(c *gin.Context, err error) {
	appErr := errs.From(err)
	c.JSON(appErr.HTTPStatus(), Response{
		Code: appErr.Code(), Message: appErr.Message(),
	})
}

type PageResp struct {
	List   any    `json:"list"`
	Total  int64  `json:"total"`
	Cursor uint64 `json:"cursor,omitempty"`
	Size   int    `json:"size,omitempty"`
}
```

### 7.4 错误码规范

| code | HTTP | 含义 |
|---|---|---|
| 0 | 200 | 成功 |
| 40001 | 400 | 参数错误 |
| 40010 | 409 | 用户名已占用 |
| 40011 | 409 | 邮箱已注册 |
| 40012 | 400 | 用户名不合法 |
| 40013 | 400 | 密码强度不足 |
| 40014 | 400 | 验证码错误或已过期 |
| 40015 | 400 | 邀请码无效 |
| 40020 | 400 | 需要验证码 |
| 40100 | 401 | 未登录 |
| 40101 | 401 | Token 过期 |
| 40300 | 403 | 无权限 |
| 40301 | 403 | 邮箱未验证 |
| 40302 | 403 | 请先完成邮箱验证 |
| 40310 | 403 | S3 签名无效 |
| 40311 | 403 | S3 时间偏差过大 |
| 40312 | 403 | S3 密钥已禁用 |
| 40400 | 404 | 资源不存在 |
| 41300 | 413 | 文件超限 |
| 41301 | 413 | 容量配额超限 |
| 41302 | 413 | 每日上传数超限 |
| 42900 | 429 | 请求过于频繁 |
| 50000 | 500 | 内部错误 |

---

## 8. API 接口设计

### 8.1 通用约定

- 基础路径：`/api/v1`
- 请求/响应：JSON（上传为 `multipart/form-data`）
- 鉴权：`Authorization: Bearer <JWT 或 API Token>`
- 时间格式：RFC3339 UTC（`2025-06-12T08:30:00Z`）
- 统一响应：`{ "code": 0, "message": "ok", "data": {...} }`
- 分页：`?page=1&size=24`（页码）或 `?cursor=xxx&size=24`（游标）

### 8.2 认证与用户

| 方法 | 路径 | 说明 | 鉴权 |
|---|---|---|---|
| GET | `/auth/captcha` | 获取图片验证码 | - |
| POST | `/auth/register` | 注册 | - |
| POST | `/auth/login` | 登录 | - |
| POST | `/auth/refresh` | 刷新 Token | - |
| POST | `/auth/logout` | 登出 | JWT |
| GET | `/auth/check-username?u=` | 用户名可用性 | - |
| GET | `/auth/check-email?e=` | 邮箱可用性 | - |
| POST | `/auth/verify-email` | 邮箱验证 | - |
| POST | `/auth/resend-verify` | 重发验证邮件 | - |
| POST | `/auth/forgot-password` | 忘记密码 | - |
| POST | `/auth/reset-password` | 重置密码 | - |
| GET | `/user/profile` | 当前用户 | JWT |
| PUT | `/user/profile` | 修改资料 | JWT |
| PUT | `/user/password` | 修改密码 | JWT |
| GET | `/user/stats` | 用量统计 | JWT |

**注册请求**

```json
POST /api/v1/auth/register
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "Passw0rd",
  "captcha_id": "a1b2c3",
  "captcha_code": "8F3K",
  "invite_code": ""
}
```

**注册响应**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "expires_in": 7200,
    "user": {
      "id": 10086,
      "username": "alice",
      "email": "alice@example.com",
      "nickname": "alice",
      "role": 0,
      "status": 2,
      "email_verified": false,
      "group_id": 1,
      "quota_bytes": 2147483648,
      "used_bytes": 0,
      "daily_limit": 200,
      "default_visibility": "public",
      "created_at": "2025-06-12T08:30:00Z"
    }
  }
}
```

### 8.3 上传

| 方法 | 路径 | 说明 | 鉴权 |
|---|---|---|---|
| POST | `/upload` | 上传单图 | JWT/Token |
| POST | `/upload/check` | 秒传检测 | JWT/Token |
| POST | `/upload/url` | 远程拉取 | JWT/Token |
| POST | `/upload/chunk/init` | 初始化分片 | JWT/Token |
| POST | `/upload/chunk/:id` | 上传分片 | JWT/Token |
| POST | `/upload/chunk/:id/complete` | 合并分片 | JWT/Token |

**上传请求**（PicGo 兼容）

```
POST /api/v1/upload
Content-Type: multipart/form-data

file:        <binary>      必填
album_id:    3             可选
tags:        "blog,cover"  可选
is_public:   true          可选
storage_id:  1             可选
```

**上传响应**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": 1024,
    "url": "https://host/i/9f86d081....png",
    "thumb_url": "https://host/t/9f86d081.../800.webp",
    "markdown": "![image](https://host/i/9f86d081....png)",
    "html": "<img src=\"https://host/i/9f86d081....png\" />",
    "bbcode": "[img]https://host/i/9f86d081....png[/img]",
    "size": 245678,
    "width": 1920,
    "height": 1080,
    "hash": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
  }
}
```

### 8.4 图片管理

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/images` | 列表（支持 `keyword` `album_id` `tag_id` `ext` `referred` `cursor`） |
| GET | `/images/:id` | 详情 |
| PATCH | `/images/:id` | 修改（名称/相册/标签/可见性） |
| DELETE | `/images/:id` | 移入回收站 |
| POST | `/images/batch` | 批量操作 |
| GET | `/images/trash` | 回收站列表 |
| POST | `/images/:id/restore` | 还原 |
| DELETE | `/images/trash` | 清空回收站 |
| GET | `/images/:id/referers` | 引用列表 |
| GET | `/images/:id/referers/summary` | 引用汇总 |
| DELETE | `/images/:id/referers/:rid` | 删除引用记录 |

### 8.5 相册、标签、Token、分享

| 方法 | 路径 | 说明 |
|---|---|---|
| GET/POST | `/albums` | 列表 / 创建 |
| GET/PUT/DELETE | `/albums/:id` | 详情 / 更新 / 删除 |
| GET/POST | `/tags` | 列表 / 创建 |
| PUT/DELETE | `/tags/:id` | 更新 / 删除 |
| GET/POST | `/tokens` | 列表 / 创建 |
| DELETE | `/tokens/:id` | 删除 |
| PUT | `/tokens/:id/status` | 启用禁用 |
| POST | `/shares` | 创建分享 |
| GET | `/shares` | 我的分享列表 |
| DELETE | `/shares/:id` | 删除分享 |
| GET | `/s/:slug` | 访问分享（可选密码） |

### 8.6 存储与 S3 凭据

| 方法 | 路径 | 说明 |
|---|---|---|
| GET/POST | `/storages` | 存储策略列表 / 创建 |
| PUT/DELETE | `/storages/:id` | 更新 / 删除 |
| POST | `/storages/:id/test` | 连通性测试 |
| GET/POST | `/s3/credentials` | S3 凭据列表 / 生成 |
| DELETE | `/s3/credentials/:id` | 删除 |

### 8.7 引用统计

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/stats/referers/top?limit=10&days=30` | TOP 域名 |
| GET | `/stats/referers/trend?days=30` | 引用趋势 |

### 8.8 管理后台

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/admin/dashboard` | 仪表盘数据 |
| GET | `/admin/users` | 用户列表 |
| PUT | `/admin/users/:id/status` | 启用/禁用 |
| PUT | `/admin/users/:id/group` | 调整用户组 |
| PUT | `/admin/users/:id/password` | 重置密码 |
| GET/POST | `/admin/groups` | 用户组列表 / 创建 |
| PUT/DELETE | `/admin/groups/:id` | 更新 / 删除 |
| GET/PUT | `/admin/settings` | 系统设置 |
| GET | `/admin/logs` | 操作日志 |
| GET/POST | `/admin/invites` | 邀请码列表 / 创建 |
| DELETE | `/admin/invites/:id` | 删除邀请码 |

### 8.9 图片直出

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/i/:hash` | 原图 |
| GET | `/t/:hash/:size` | 缩略图 |
| GET | `/i/:hash?w=&h=&fit=&q=&format=&blur=&gray=` | 动态处理 |

**动态处理参数**

| 参数 | 含义 | 示例 |
|---|---|---|
| `w` / `h` | 目标宽/高 | `?w=800` |
| `fit` | cover / contain / fill | `?w=200&h=200&fit=cover` |
| `q` | 质量 1-100 | `?q=75` |
| `format` | webp / jpeg / png / avif | `?format=webp` |
| `blur` | 高斯模糊半径 | `?blur=10` |
| `gray` | 灰度 0/1 | `?gray=1` |

### 8.10 S3 兼容网关

```
Base URL: https://{host}/s3

PUT    /s3/{bucket}/{key}              上传对象
GET    /s3/{bucket}/{key}              下载对象
HEAD   /s3/{bucket}/{key}              获取元信息
DELETE /s3/{bucket}/{key}              删除对象
GET    /s3/{bucket}?list-type=2        列举对象
POST   /s3/{bucket}/{key}?uploads      创建分片上传
PUT    /s3/{bucket}/{key}?partNumber=N&uploadId=X
POST   /s3/{bucket}/{key}?uploadId=X   完成分片
DELETE /s3/{bucket}/{key}?uploadId=X   取消分片
GET    /s3/{bucket}?location           GetBucketLocation
HEAD   /s3/{bucket}                    HeadBucket
```

**PicGo 配置示例**

```json
{
  "endpoint": "https://your-host/s3",
  "bucket": "pichub",
  "region": "us-east-1",
  "accessKeyID": "PKIA1B2C3D4E5F6G7H8I",
  "secretAccessKey": "xxxx",
  "pathStyle": true,
  "customDomain": "https://your-host/s3/pichub"
}
```

### 8.11 探针与文档

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/healthz` | 存活探针 |
| GET | `/readyz` | 就绪探针（检查 DB / Redis） |
| GET | `/metrics` | Prometheus 指标（可关闭） |
| GET | `/swagger/index.html` | API 文档 |

---

## 9. 前端设计

### 9.1 设计理念

**克制、留白、一致。**

| 原则 | 具体表现 |
|---|---|
| 克制用色 | 90% 中性色，主色仅用于主要操作与状态强调 |
| 充分留白 | 8px 基准间距，卡片内边距 ≥ 20px |
| 清晰层级 | 字重 + 字号 + 灰度三层对比，而非依赖边框 |
| 一致体验 | 同功能同组件，不发明一次性样式 |
| 优雅动效 | 150–250ms，`cubic-bezier(0.4,0,0.2,1)` |
| 图像优先 | 图库/预览页让图片成为主角，UI 悬浮淡入 |

### 9.2 设计令牌

```scss
// styles/tokens/_color.scss
:root {
  --ph-primary:          #409EFF;
  --ph-primary-hover:    #66B1FF;
  --ph-primary-active:   #3A8EE6;
  --ph-primary-light-3:  #79BBFF;
  --ph-primary-light-5:  #A0CFFF;
  --ph-primary-light-7:  #C6E2FF;
  --ph-primary-light-8:  #D9ECFF;
  --ph-primary-light-9:  #ECF5FF;

  --ph-text-primary:     #1F2329;
  --ph-text-regular:     #4E5969;
  --ph-text-secondary:   #86909C;
  --ph-text-placeholder: #C9CDD4;
  --ph-border:           #E5E6EB;
  --ph-border-light:     #F2F3F5;
  --ph-fill:             #F7F8FA;
  --ph-fill-card:        #FFFFFF;
  --ph-fill-hover:       #F2F3F5;

  --ph-success:          #00B42A;
  --ph-warning:          #FF7D00;
  --ph-danger:           #F53F3F;
  --ph-info:             #86909C;

  --ph-radius-sm:        4px;
  --ph-radius-md:        8px;
  --ph-radius-lg:        12px;
  --ph-radius-xl:        16px;
  --ph-radius-full:      9999px;

  --ph-shadow-sm:  0 1px 2px rgba(0,0,0,0.04);
  --ph-shadow-md:  0 4px 12px rgba(0,0,0,0.08);
  --ph-shadow-lg:  0 8px 24px rgba(0,0,0,0.12);
}

html.dark {
  --ph-text-primary:   #E5E6EB;
  --ph-text-regular:   #C9CDD4;
  --ph-text-secondary: #86909C;
  --ph-border:         #333335;
  --ph-border-light:   #2A2A2C;
  --ph-fill:           #17171A;
  --ph-fill-card:      #1F1F23;
  --ph-fill-hover:     #2A2A2C;
}
```

```scss
// styles/tokens/_typography.scss
:root {
  --ph-font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
                    'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue',
                    Helvetica, Arial, sans-serif;
  --ph-font-mono: 'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace;
}

body {
  font-family: var(--ph-font-family);
  font-size: 14px;
  line-height: 1.6;
  color: var(--ph-text-regular);
  -webkit-font-smoothing: antialiased;
}
```

```scss
// styles/tokens/_spacing.scss
:root {
  --ph-space-1:  4px;
  --ph-space-2:  8px;
  --ph-space-3:  12px;
  --ph-space-4:  16px;
  --ph-space-5:  20px;
  --ph-space-6:  24px;
  --ph-space-8:  32px;
  --ph-space-10: 40px;
  --ph-space-12: 48px;
  --ph-space-16: 64px;
}
```

### 9.3 Element Plus 主题定制

```scss
// styles/element/index.scss
@forward 'element-plus/theme-chalk/src/common/var.scss' with (
  $colors: (
    'primary': ('base': #409EFF),
    'success': ('base': #00B42A),
    'warning': ('base': #FF7D00),
    'danger':  ('base': #F53F3F),
    'info':    ('base': #86909C),
  ),
  $text-color: (
    'primary':   #1F2329,
    'regular':   #4E5969,
    'secondary': #86909C,
    'placeholder': #C9CDD4,
  ),
  $border-color: (
    '':        #E5E6EB,
    'light':   #F2F3F5,
    'lighter': #F7F8FA,
  ),
  $bg-color: (
    '':        #FFFFFF,
    'page':    #F7F8FA,
    'overlay': #FFFFFF,
  ),
  $border-radius: (
    'base':   8px,
    'small':  4px,
    'round':  20px,
    'circle': 100%,
  ),
);
```

```ts
// vite.config.ts
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

export default defineConfig({
  plugins: [
    vue(),
    AutoImport({ resolvers: [ElementPlusResolver({ importStyle: 'sass' })] }),
    Components({ resolvers: [ElementPlusResolver({ importStyle: 'sass' })] }),
  ],
  css: {
    preprocessorOptions: {
      scss: { additionalData: `@use "@/styles/element/index.scss" as *;` },
    },
  },
})
```

### 9.4 路由表

```ts
const routes = [
  { path: '/login',           component: () => import('@/views/auth/LoginView.vue'),        meta: { layout: 'blank', public: true } },
  { path: '/register',        component: () => import('@/views/auth/RegisterView.vue'),     meta: { layout: 'blank', public: true } },
  { path: '/verify-email',    component: () => import('@/views/auth/VerifyEmailView.vue'),  meta: { layout: 'blank', public: true } },
  { path: '/forgot-password', component: () => import('@/views/auth/ForgotPasswordView.vue'), meta: { layout: 'blank', public: true } },
  { path: '/s/:slug',         component: () => import('@/views/share/ShareView.vue'),       meta: { layout: 'blank', public: true } },

  { path: '/',         component: () => import('@/views/workspace/WorkspaceView.vue'),  meta: { auth: true } },
  { path: '/gallery',  component: () => import('@/views/gallery/GalleryView.vue'),      meta: { auth: true } },
  { path: '/albums',   component: () => import('@/views/albums/AlbumListView.vue'),     meta: { auth: true } },
  { path: '/albums/:id', component: () => import('@/views/albums/AlbumDetailView.vue'), meta: { auth: true } },
  { path: '/trash',    component: () => import('@/views/trash/TrashView.vue'),          meta: { auth: true } },
  { path: '/tokens',   component: () => import('@/views/tokens/TokenView.vue'),         meta: { auth: true } },
  { path: '/settings', component: () => import('@/views/settings/SettingsView.vue'),    meta: { auth: true } },
  { path: '/stats',    component: () => import('@/views/stats/StatsView.vue'),          meta: { auth: true } },

  { path: '/admin',           component: () => import('@/views/admin/DashboardView.vue'), meta: { auth: true, admin: true } },
  { path: '/admin/users',     component: () => import('@/views/admin/UsersView.vue'),     meta: { auth: true, admin: true } },
  { path: '/admin/groups',    component: () => import('@/views/admin/GroupsView.vue'),    meta: { auth: true, admin: true } },
  { path: '/admin/storages',  component: () => import('@/views/admin/StoragesView.vue'),  meta: { auth: true, admin: true } },
  { path: '/admin/settings',  component: () => import('@/views/admin/SettingsView.vue'),  meta: { auth: true, admin: true } },
  { path: '/admin/logs',      component: () => import('@/views/admin/LogsView.vue'),      meta: { auth: true, admin: true } },
  { path: '/admin/invites',   component: () => import('@/views/admin/InvitesView.vue'),   meta: { auth: true, admin: true } },
]
```

### 9.5 响应式断点

| 名称 | 范围 | 目标 |
|---|---|---|
| `xs` | < 768px | 手机 |
| `sm` | 768–991px | 小平板 |
| `md` | 992–1199px | 平板 |
| `lg` | 1200–1919px | 笔记本 |
| `xl` | ≥ 1920px | 大屏 |

```ts
// composables/useDevice.ts
import { useBreakpoints } from '@vueuse/core'
export const useDevice = () => {
  const bp = useBreakpoints({ mobile: 0, tablet: 768, desktop: 1200 })
  return {
    isMobile:  bp.smaller('tablet'),
    isTablet:  bp.between('tablet', 'desktop'),
    isDesktop: bp.greaterOrEqual('desktop'),
  }
}
```

### 9.6 布局形态

**桌面端（≥1200px）**

```
┌──────────────────────────────────────────────────┐
│  TopBar 64px   Logo · 搜索 · 上传 · 头像           │
├────────┬─────────────────────────────────────────┤
│Sidebar │                                         │
│ 220px  │            Content (max 1400px)          │
│        │            padding: 32px                  │
└────────┴─────────────────────────────────────────┘
```

**平板端（768–1199px）**

- Sidebar 折叠为 64px 图标栏，hover 展开浮层
- Content padding 24px
- 图库 3 列

**手机端（<768px）**

```
┌─────────────────────────────────┐
│  TopBar 56px  汉堡 · 标题 · 搜索  │
├─────────────────────────────────┤
│                                 │
│  Content padding: 16px          │
│                                 │
├─────────────────────────────────┤
│  TabBar 56px 工作台 图库 相册 我  │
└─────────────────────────────────┘
```

### 9.7 关键页面视觉

**上传工作台**

```
┌────────────────────────────────────────────────┐
│  上传图片                          [设置参数 ⚙]  │
├────────────────────────────────────────────────┤
│      ┌──────────────────────────────────┐      │
│      │         ☁ 大图标 (64px)           │      │
│      │    拖拽图片到这里，或点击上传       │      │
│      │  支持 JPG PNG GIF WebP · 单张10MB │      │
│      └──────────────────────────────────┘      │
│      虚线边框 2px 圆角 12px 背景 #FAFBFC         │
├────────────────────────────────────────────────┤
│  上传队列 (3)                                   │
│  ┌────────┐ ┌────────┐ ┌────────┐              │
│  │ 缩略图  │ │ 缩略图  │ │ 缩略图  │              │
│  │ ▓▓▓ 80%│ │ ✓ 完成 │ │ ✗ 失败 │              │
│  └────────┘ └────────┘ └────────┘              │
├────────────────────────────────────────────────┤
│  上传完成                                       │
│  [缩略图] photo-2025.png · 1.2MB · 1920×1080    │
│  [URL] [Markdown] [HTML] [BBCode]  ← 复制按钮组  │
└────────────────────────────────────────────────┘
```

**图库页**

- 网格：桌面 `repeat(auto-fill, minmax(220px, 1fr))`，间距 16px
- 卡片圆角 12px，白底，1px 浅边框，hover 上浮 2px + 阴影
- 缩略图固定 1:1 比例，`object-fit: cover`
- hover 遮罩 `rgba(0,0,0,0.5)` 淡入，居中显示操作图标

**登录/注册页**

- 桌面：左右分栏，左侧品牌区（渐变 `linear-gradient(135deg, #409EFF 0%, #667EEA 100%)`），右侧表单区
- 表单最大宽 380px，垂直居中，输入框 44px 高
- 手机：单栏，品牌区压缩为顶部 Logo + slogan

### 9.8 动效规范

| 场景 | 时长 | 缓动 | 效果 |
|---|---|---|---|
| 按钮 hover | 150ms | ease-out | 背景色 + 上移 1px |
| 卡片 hover | 200ms | cubic-bezier(0.4,0,0.2,1) | 上移 2px + 阴影 |
| 抽屉/弹窗 | 250ms | cubic-bezier(0.4,0,0.2,1) | 滑入/淡入 |
| 下拉菜单 | 150ms | ease-out | 淡入 + 缩放 0.96→1 |
| 页面切换 | 200ms | ease-in-out | 淡入淡出 |

**红线**：不用弹跳/旋转/翻转；不超过 400ms；尊重 `prefers-reduced-motion`。

### 9.9 组件清单

**通用组件**

| 组件 | 说明 |
|---|---|
| `AppTopBar` | 顶部栏 |
| `AppSidebar` | 侧边栏 |
| `AppTabBar` | 手机底部标签栏 |
| `EmptyState` | 统一空状态 |
| `LoadingSkeleton` | 骨架屏 |
| `PageHeader` | 页面标题栏 |
| `CaptchaField` | 验证码输入 + 图片 |
| `CopyButton` | 复制按钮（带 Toast） |
| `ThemeToggle` | 亮暗模式切换 |

**上传组件**

| 组件 | 说明 |
|---|---|
| `UploadDropzone` | 拖拽/点击/粘贴上传区 |
| `UploadQueue` | 上传队列 |
| `UploadCard` | 单张上传卡片（进度/状态） |
| `UploadParams` | 上传参数面板（相册/标签/可见性） |

**图片组件**

| 组件 | 说明 |
|---|---|
| `ImageGrid` | 响应式网格 |
| `ImageCard` | 图片卡片 |
| `ImageDetailDrawer` | 详情抽屉（桌面） / 全屏页（手机） |
| `ImageLinkList` | 多格式外链复制 |
| `RefererList` | 引用来源列表 |

### 9.10 移动端专项

| 项 | 方案 |
|---|---|
| 上传 | 支持调用相机 `<input type="file" accept="image/*" capture="environment">` |
| 手势 | 图片预览支持双指缩放、左右滑动 |
| 安全区 | `env(safe-area-inset-bottom)` 适配底部标签栏 |
| 图片懒加载 | `loading="lazy"` + `srcset` 多倍图 |
| 深色模式 | 跟随系统 |
| 弱网 | 上传超时 60s，失败自动重试 2 次 |
| 分页 | 无限滚动（`IntersectionObserver`） |

---

## 10. 前后端一体化嵌入

### 10.1 构建流程

```
web/ 源码 → vite build → web/dist → 复制到 server/internal/web/dist
         → go build -tags=embed → 单二进制（含前端）
```

### 10.2 Embed 实现

**`internal/web/embed.go`**

```go
//go:build embed

package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS 返回前端静态资源文件系统
func FS() (fs.FS, bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, false
	}
	return sub, true
}
```

**`internal/web/noembed.go`**

```go
//go:build !embed

package web

import "io/fs"

func FS() (fs.FS, bool) { return nil, false }
```

> ⚠️ `go:embed` 不能嵌入空目录，仓库必须保留 `internal/web/dist/.gitkeep`。

### 10.3 SPA Fallback

**`internal/router/static.go`**

```go
package router

const (
	cacheImmutable = "public, max-age=31536000, immutable"
	cacheNoCache   = "no-cache"
)

func registerStatic(r *gin.Engine, assets fs.FS) {
	if assets == nil {
		return
	}
	fileServer := http.FileServer(http.FS(assets))

	r.NoRoute(func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		// API / S3 / 图片路径不 fallback
		if isReservedPath(reqPath) {
			c.JSON(http.StatusNotFound, gin.H{
				"code": 40400, "message": "resource not found",
			})
			return
		}

		// 命中真实静态文件
		clean := strings.TrimPrefix(reqPath, "/")
		if clean != "" {
			if f, err := assets.Open(clean); err == nil {
				_ = f.Close()
				if strings.HasPrefix(reqPath, "/assets/") {
					c.Header("Cache-Control", cacheImmutable)
				}
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// SPA fallback → index.html
		index, err := fs.ReadFile(assets, "index.html")
		if err != nil {
			c.String(500, "frontend not built")
			return
		}
		c.Header("Cache-Control", cacheNoCache)
		c.Data(200, "text/html; charset=utf-8", index)
	})
}

func isReservedPath(p string) bool {
	for _, pre := range []string{"/api/", "/s3/", "/swagger/", "/i/", "/t/"} {
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return p == "/healthz" || p == "/readyz" || p == "/metrics"
}
```

### 10.4 Vite 配置

```ts
// web/vite.config.ts
export default defineConfig({
  plugins: [vue(), AutoImport({...}), Components({...})],
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    emptyOutDir: true,
    sourcemap: false,
    rollupOptions: {
      output: {
        entryFileNames: 'assets/[name]-[hash].js',
        chunkFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash].[ext]',
        manualChunks: {
          vue:    ['vue', 'vue-router', 'pinia'],
          element:['element-plus', '@element-plus/icons-vue'],
          charts: ['echarts', 'vue-echarts'],
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api':     { target: 'http://localhost:8080', changeOrigin: true },
      '/i':       { target: 'http://localhost:8080', changeOrigin: true },
      '/t':       { target: 'http://localhost:8080', changeOrigin: true },
      '/s3':      { target: 'http://localhost:8080', changeOrigin: true },
      '/swagger': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
  base: '/',
})
```

### 10.5 Makefile

```makefile
.PHONY: dev-api dev-web build-web build build-docker test lint clean

BINARY := pichub
DIST   := internal/web/dist

dev-api:
	cd server && air

dev-web:
	cd web && npm run dev

build-web:
	cd web && npm ci && npm run build
	rm -rf server/$(DIST) && mkdir -p server/$(DIST) && cp -r web/dist/. server/$(DIST)/

build: build-web
	cd server && CGO_ENABLED=0 go build -tags=embed -trimpath \
		-ldflags="-s -w -X main.version=$(shell git describe --tags --always)" \
		-o ../bin/$(BINARY) ./cmd/pichub

build-docker:
	docker build -t pichub/pichub:$(shell git describe --tags --always) .

test:
	cd server && go test ./... -race -cover

lint:
	cd server && golangci-lint run ./...
	cd web && npm run lint

clean:
	rm -rf bin web/dist server/$(DIST)/*
	touch server/$(DIST)/.gitkeep
```

### 10.6 开发模式

**方式 A：前后端分离（推荐）**

```bash
# 终端 1：后端
cd server && air          # 或 go run -tags=noembed ./cmd/pichub

# 终端 2：前端 HMR
cd web && npm run dev     # http://localhost:5173
```

**方式 B：一体化调试**

```bash
make build && ./bin/pichub -c config.yaml
```

---

## 11. Docker 部署

### 11.1 Dockerfile（三阶段）

```dockerfile
# ---------- Stage 1: 前端构建 ----------
FROM node:20-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---------- Stage 2: 后端构建 ----------
FROM golang:1.22-alpine AS api-builder
WORKDIR /src
RUN apk add --no-cache git make
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
COPY --from=web-builder /web/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -tags=embed -trimpath \
    -ldflags="-s -w" -o /out/pichub ./cmd/pichub

# ---------- Stage 3: 运行镜像 ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -u 10001 pichub
WORKDIR /app
COPY --from=api-builder /out/pichub /app/pichub
COPY server/configs/config.example.yaml /app/config.example.yaml
RUN mkdir -p /app/data/uploads /app/data/thumbs /app/data/cache /app/data/logs && \
    chown -R pichub:pichub /app
USER pichub
EXPOSE 8080
ENV TZ=UTC
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/pichub", "-c", "/app/config.yaml"]
```

### 11.2 docker-compose.yml

```yaml
services:
  pichub:
    image: pichub/pichub:latest
    container_name: pichub
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./data/uploads:/app/data/uploads
      - ./data/thumbs:/app/data/thumbs
      - ./data/cache:/app/data/cache
      - ./data/logs:/app/data/logs
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks: [pichub]

  mysql:
    image: mysql:8.0.38
    restart: unless-stopped
    command:
      - --character-set-server=utf8mb4
      - --collation-server=utf8mb4_0900_ai_ci
      - --default-time-zone=+00:00
      - --max_connections=200
      - --innodb-buffer-pool-size=512M
    environment:
      MYSQL_ROOT_PASSWORD_FILE: /run/secrets/mysql_root
      MYSQL_PASSWORD_FILE: /run/secrets/mysql_pichub
      MYSQL_DATABASE: pichub
      MYSQL_USER: pichub
    secrets:
      - mysql_root
      - mysql_pichub
    volumes:
      - mysql-data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "--silent"]
      interval: 10s
      timeout: 5s
      retries: 10
    networks: [pichub]

  redis:
    image: redis:7.2-alpine
    restart: unless-stopped
    command:
      - redis-server
      - --appendonly
      - "yes"
      - --maxmemory
      - 256mb
      - --maxmemory-policy
      - allkeys-lru
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5
    networks: [pichub]

volumes:
  mysql-data:
  redis-data:

secrets:
  mysql_root:
    file: ./secrets/mysql_root.txt
  mysql_pichub:
    file: ./secrets/mysql_pichub.txt

networks:
  pichub:
    driver: bridge
```

> 说明：MySQL 官方镜像读取密码用的是文件方式（`_FILE`），属于容器自身机制，不违反「应用层不用环境变量」的约定。

### 11.3 一键部署脚本

```bash
#!/bin/bash
# deploy/install.sh
set -e

mkdir -p data/{uploads,thumbs,cache,logs} secrets

# 生成配置
if [ ! -f config.yaml ]; then
  cp server/configs/config.yaml ./config.yaml
fi

# 生成密钥
JWT=$(openssl rand -hex 32)
SECRET=$(openssl rand -base64 32)
DBPASS=$(openssl rand -base64 24 | tr -d '/+=' | head -c 24)
ROOTPASS=$(openssl rand -base64 24 | tr -d '/+=' | head -c 24)

sed -i "s|CHANGE_ME_JWT_SECRET_AT_LEAST_32_CHARS|${JWT}|" config.yaml
sed -i "s|CHANGE_ME_SECRET_KEY_BASE64_32_BYTES|${SECRET}|" config.yaml
sed -i "s|CHANGE_ME_DB_PASSWORD|${DBPASS}|" config.yaml

echo "$DBPASS"  > secrets/mysql_pichub.txt
echo "$ROOTPASS" > secrets/mysql_root.txt
chmod 600 config.yaml secrets/*.txt

echo "配置已生成，请编辑 config.yaml 中的 site.url 与 mail.* 后执行："
echo "  docker compose -f deploy/docker-compose.yml up -d"
```

### 11.4 升级与备份

| 操作 | 命令 |
|---|---|
| 升级 | `docker compose pull && docker compose up -d` |
| 数据库备份 | `docker compose exec mysql mysqldump -upichub -p --single-transaction --set-gtid-purged=OFF pichub > backup_$(date +%F).sql` |
| 上传目录备份 | `tar -czf uploads_$(date +%F).tar.gz ./data/uploads` |
| 恢复 | 先恢复 SQL，再解压 uploads |

---

## 12. 关键特性实现

### 12.1 秒传（Hash 去重）

**流程**

```
前端选择文件
  → Web Worker 计算 SHA-256（crypto.subtle.digest）
  → POST /api/v1/upload/check { hash, size, ext }
      ├─ 命中：直接返回既有记录 → 结束
      └─ 未命中：POST /api/v1/upload (multipart)
```

**服务端去重**：

- 唯一索引 `uk_user_hash_alive(user_id, hash, deleted_flag)`
- `dedup_scope=user`：仅本用户内去重
- `dedup_scope=global`：全站去重，物理文件引用计数 `ref_count`，归零才删存储

### 12.2 图片验证码

**方案**：`github.com/mojocn/base64Captcha`

- 类型：数字 + 字母 4 位，可切换算术/中文
- 存储：Redis `captcha:{id}`，TTL 300s，**一次性消费**
- 触发场景：注册（强制）、登录（失败 2 次后）、找回密码（强制）、重发验证邮件（强制）、修改密码（强制）

**接口**

```
GET /api/v1/auth/captcha
→ { "code":0, "data": {
    "captcha_id": "a1b2c3",
    "image_base64": "data:image/png;base64,...",
    "expires_in": 300
  } }
```

**校验**：Redis 取 → 比对（大小写不敏感）→ 无论成败立即 DEL → 失败返回 `40014`。

### 12.3 邮箱验证

**状态机**

```
注册 → status=2, email_verified=0
     → 发送验证邮件（token 24h 有效）
     → 用户点击链接 → GET /verify-email?token=xxx
         ├─ 成功 → status=1, email_verified=1
         └─ 过期 → 展示「重新发送」
```

- **force 模式**：未验证不可登录，返回 `40301`
- **soft 模式**：可登录但禁止上传，返回 `40302`
- **降级保护**：`require_email_verify=force` 但 SMTP 未配置时，启动自动降级为 `soft` 并打印警告

**邮件模板**：`verify.html` / `reset.html` / `change.html`，异步队列发送，失败重试 3 次。

### 12.4 图片引用统计

**采集链路**

```
GET /i/xxx.png
  Referer: https://blog.example.com/post/1
    ↓ 过滤（空 Referer / 本站域名 / 爬虫 UA）
    ↓ 异步写入 Redis 缓冲
    HINCRBY ref:buf:{yyyyMMdd} "{image_id}|{referer_hash}" 1
    SETEX   ref:meta:{referer_hash} 172800 "{referer_url}"
    ↓ cron 每 60 秒批量刷盘
    INSERT ... ON DUPLICATE KEY UPDATE hit_count = hit_count + VALUES(hit_count)
```

**异步实现**：带缓冲 channel（cap=10000）+ 单 goroutine 消费；channel 满时丢弃并计数，保证图片直出永不阻塞。

**API**

| 方法 | 路径 |
|---|---|
| GET | `/images/:id/referers` |
| GET | `/images/:id/referers/summary` |
| GET | `/stats/referers/top?limit=10&days=30` |
| DELETE | `/images/:id/referers/:rid` |

**隐私**：仅记录 Referer URL 与域名，不记录访客 IP / UA / Cookie；单图上限 200 条。

### 12.5 S3 兼容网关（SigV4）

**AK/SK 生成**

```
Access Key ID:     PKIA1B2C3D4E5F6G7H8I      （20 位，前缀 PK）
Secret Access Key: k9Xm2...                   （40 位随机串，仅展示一次）
Region:            us-east-1                  （占位，不校验）
```

**校验流程**

```
1. 解析 Authorization 头：
   AWS4-HMAC-SHA256 Credential=PKIA.../20250612/us-east-1/s3/aws4_request,
   SignedHeaders=host;x-amz-content-sha256;x-amz-date,
   Signature=xxxx
2. 查 s3_credentials → 取 secret（AES-GCM 解密）
3. 重算 CanonicalRequest + StringToSign + SigningKey + Signature
4. hmac.Equal 常量时间比对
5. 校验 x-amz-date 偏差 ≤ 15 分钟
6. 校验签名未重放（Redis: sha256(signature)，TTL 15min）
7. 注入 user_id 到 context
```

**上传后处理**：与普通上传完全一致（Hash → 校验 → 存储 → 缩略图 → 入库 → 配额）。

**PicGo 配置**

```json
{
  "endpoint": "https://your-host/s3",
  "bucket": "pichub",
  "region": "us-east-1",
  "accessKeyID": "PKIA1B2C3D4E5F6G7H8I",
  "secretAccessKey": "xxxx",
  "pathStyle": true,
  "customDomain": "https://your-host/s3/pichub"
}
```

### 12.6 本地存储

**路径生成**

```
模板：{prefix}/{yyyy}/{MM}/{dd}/{hash2}/{hash}.{ext}
示例：uploads/2025/06/12/9f/9f86d081....png
```

**原子写入**：先写 `.tmp`，`fsync` 后 `rename`，避免半成品被读取。

**路径穿越防护**：

```go
abs := filepath.Join(rootPath, filepath.FromSlash(key))
if !strings.HasPrefix(abs, filepath.Clean(rootPath)+string(filepath.Separator)) {
	return "", ErrInvalidKey
}
```

**图片直出**：使用 `http.ServeContent` 而非 `io.Copy`，免费获得 Range / ETag / 条件请求支持。

**本地 → 对象存储迁移工具**

```bash
./pichub migrate storage --from local --to s3-main --dry-run
./pichub migrate storage --from local --to s3-main --concurrency 8 [--delete-source]
```

### 12.7 存储策略优先级

```
1. 用户显式指定（上传请求 storage_id）
2. 用户默认策略（users.storage_config_id）
3. 系统默认策略（storage_configs.is_default = 1）
4. YAML 兜底（storage.default_driver = local）  ← 保底
```

**启动初始化**：查 DB 是否存在 `driver=local` 的策略，不存在则自动创建（is_default=1），配置从 YAML 读取。

### 12.8 配额校验（原子）

```sql
UPDATE users
SET used_bytes = used_bytes + ?
WHERE id = ? AND (quota_bytes = 0 OR used_bytes + ? <= quota_bytes);
-- 通过 RowsAffected 判断是否超额
```

上传前校验：

1. 单文件大小 ≤ `max_file_size`（用户级或组级）
2. 每日张数 ≤ `daily_limit`（`upload_counters`）
3. 总容量原子校验

---

## 13. 开发里程碑

| 阶段 | 周期 | 交付内容 |
|---|---|---|
| **M1 基础框架** | 1.5 周 | 项目骨架、YAML 配置加载（含密钥自动生成）、MySQL 迁移、JWT 鉴权、注册/登录、图片验证码、邮箱验证全链路、IP 限流、Docker Compose、三端布局框架、embed 机制打通、SPA fallback、Element Plus 主题 |
| **M2 核心闭环** | 1.5 周 | 本地存储实现、上传（点击/拖拽/粘贴/相机）、图片列表/详情/删除、外链生成、缩略图、图库页、上传工作台 |
| **M3 资产管理** | 1 周 | 相册、标签、搜索筛选、批量操作、回收站、个人设置、配额展示 |
| **M4 扩展能力** | 1.5 周 | API Token、分享页、对象存储驱动、动态图片处理、防盗链、用户组配额、迁移工具 |
| **M5 引用统计 + 管理后台** | 1.5 周 | 引用采集链路、引用展示、仪表盘、用户管理、用户组、系统设置、存储策略、操作日志、邀请码 |
| **M6 S3 兼容网关** | 1.5 周 | SigV4 校验、PutObject/GetObject/Head/Delete/ListObjectsV2/分片上传、AK/SK 管理、PicGo 实测联调 |
| **M7 打磨发布** | 1 周 | 秒传、限流压测、单元/集成测试、Docker 一键部署、API 文档、移动端真机回归、Bug 修复 |

**总计约 11 周**（1 后端 + 1 前端并行）。

---

## 14. 验收清单

### 14.1 功能验收

| # | 检查项 |
|---|---|
| 1 | 首个注册用户自动成为管理员 |
| 2 | 注册强制邮箱验证，未验证不可登录 |
| 3 | 登录失败 2 次后出现图片验证码 |
| 4 | 上传支持点击/拖拽/粘贴/移动端相机 |
| 5 | 上传后返回 URL/Markdown/HTML/BBCode 四种格式 |
| 6 | 图片默认公开，可通过外链直接访问 |
| 7 | 私有图需签名访问，无签名返回 403 |
| 8 | 相册、标签、搜索、批量操作可用 |
| 9 | 删除进回收站，30 天内可还原 |
| 10 | API Token 可创建，PicGo 上传成功 |
| 11 | S3 网关可用，PicGo S3 插件上传成功且入库 |
| 12 | 分享页支持密码与过期 |
| 13 | 引用统计正确记录外部 Referer |
| 14 | 管理员可管理用户、用户组、系统设置 |
| 15 | 未开启邮件功能时，邮箱验证自动降级为 off |

### 14.2 技术验收

| # | 检查项 |
|---|---|
| 1 | `go build -tags=embed` 产出单二进制，≤ 60 MB |
| 2 | 单二进制启动后访问 `:8080` 得到完整前端 |
| 3 | 刷新任意前端路由不 404 |
| 4 | `/api/v1/*` 不存在时返回 JSON 404 而非 HTML |
| 5 | `go build -tags=noembed` 可独立跑后端，前端走 Vite |
| 6 | `handler` 包中无 `gorm` 引用 |
| 7 | `service` 包中无 `gin` 引用 |
| 8 | `model` 包无任何 internal 依赖 |
| 9 | 仓储接口定义在 `service/port.go` |
| 10 | 代码中 `grep -r "os.Getenv" server/` 无业务命中 |
| 11 | `docker-compose.yml` 中 pichub 服务无 `environment` 段 |
| 12 | 配置文件权限非 0600 时启动打印警告 |
| 13 | 首次启动自动生成密钥并回写 YAML |
| 14 | 启动日志打印实际加载的配置文件绝对路径 |
| 15 | 未知配置字段启动时报错 |
| 16 | `data/uploads/` 自动创建，上传成功落盘 |
| 17 | 未配置对象存储时，上传/访问/缩略图全链路正常 |
| 18 | `golangci-lint run` 无告警 |
| 19 | 前端资源响应头含 `immutable`，index.html 含 `no-cache` |
| 20 | `docker compose up -d` 后仅 3 容器，前端由 pichub 提供 |

### 14.3 视觉验收

| # | 检查项 |
|---|---|
| 1 | 所有页面使用统一色彩变量，无硬编码色值 |
| 2 | 所有间距为 4px 倍数 |
| 3 | 每个页面最多一个实心主色按钮 |
| 4 | 卡片默认无阴影，hover 才浮起 |
| 5 | 所有过渡 150–250ms |
| 6 | 空状态、加载态、错误态三态齐全 |
| 7 | 桌面/平板/手机三端布局正常，无横向滚动条 |
| 8 | 暗色模式所有页面可读 |
| 9 | 键盘 Tab 可完成登录与上传流程 |
| 10 | 对比度符合 WCAG AA |
| 11 | 移动端可点击区域 ≥ 44px |
| 12 | 图片懒加载生效，滚动 FPS ≥ 55 |

---

## 附录

### 附录 A：给 AI 的实现顺序建议

请按以下顺序实现，每完成一步可运行验证：

**第 1 步：项目骨架**

1. 创建目录结构
2. `go.mod` + 依赖
3. `internal/config`（YAML 加载 + 校验 + 密钥生成）
4. `internal/logger`
5. `internal/errs`
6. `cmd/pichub/main.go`（可启动，仅打印日志）

**第 2 步：基础设施**

7. `internal/bootstrap/db.go`（MySQL 连接）
8. `migrations/000001_init.up.sql`（本文档 §5）
9. `internal/bootstrap/migrate.go`
10. `internal/bootstrap/redis.go`
11. `internal/model/*`（GORM 实体）
12. `internal/repository/*`（数据访问）

**第 3 步：认证**

13. `internal/cache/captcha.go`
14. `internal/auth/*`（JWT、密码、Token、签名）
15. `internal/service/auth_service.go`
16. `internal/handler/auth_handler.go`
17. `internal/middleware/auth.go`
18. 前端登录/注册页

**第 4 步：上传与本地存储**

19. `internal/storage/storage.go`（接口）
20. `internal/storage/local/local.go`
21. `internal/storage/manager.go`
22. `internal/imageproc/*`（缩略图）
23. `internal/service/upload_service.go`
24. `internal/handler/upload_handler.go`
25. `internal/router/image.go`（/i /t 直出）
26. 前端上传工作台

**第 5 步：图片管理**

27. `internal/service/image_service.go`
28. `internal/handler/image_handler.go`
29. 前端图库页 + 详情抽屉

**第 6 步：嵌入与部署**

30. `internal/web/embed.go` + `noembed.go`
31. `internal/router/static.go`
32. `Dockerfile` + `docker-compose.yml`
33. `Makefile`

**第 7 步：扩展能力**

34. 相册、标签、Token、分享
35. 对象存储驱动
36. 动态图片处理

**第 8 步：引用统计与管理后台**

37. `internal/cache/referer_buffer.go`
38. `internal/cron/referer_aggregate.go`
39. `internal/service/referer_service.go`
40. 管理后台各页面

**第 9 步：S3 兼容网关**

41. `internal/s3gw/sigv4.go`
42. `internal/s3gw/handler.go`
43. `internal/s3gw/multipart.go`
44. 前端 S3 凭据管理页

**第 10 步：打磨**

45. 秒传
46. 限流
47. 单元/集成测试
48. 移动端适配回归
49. 文档

### 附录 B：关键第三方依赖速查

```
# Go
github.com/gin-gonic/gin                v1.10.0
gorm.io/gorm                            v1.25.10
gorm.io/driver/mysql                    v1.5.7
github.com/redis/go-redis/v9            v9.5.3
github.com/golang-migrate/migrate/v4    v4.17.1
github.com/golang-jwt/jwt/v5            v5.2.1
github.com/spf13/viper                  v1.19.0
go.uber.org/zap                         v1.27.0
github.com/robfig/cron/v3               v3.0.1
github.com/mojocn/base64Captcha         v1.3.6
github.com/disintegration/imaging       v1.6.2
github.com/chai2010/webp                v1.1.1
github.com/aws/aws-sdk-go-v2            v1.30.0
github.com/aws/aws-sdk-go-v2/service/s3 v1.58.0
github.com/swaggo/gin-swagger           v1.6.0
golang.org/x/crypto                     v0.25.0
github.com/go-playground/validator/v10  v10.22.0

# npm
vue@^3.4
element-plus@^2.7
pinia@^2
vue-router@^4
axios@^1.7
vite@^5
typescript@^5
unplugin-auto-import
unplugin-vue-components
@vueuse/core@^10
vue-virtual-scroller@^1
echarts@^5
vue-echarts@^7
sass
unocss
```

---

**文档结束**

本文档为 PicHub 图床系统的完整设计说明，涵盖产品决策、技术选型、数据库、配置、分层架构、API、前端设计、嵌入方案、部署与关键特性实现，可直接作为 AI 实现依据。实现过程中如遇本文档未覆盖的细节，优先遵循「克制、简洁、一致」原则进行合理决策。