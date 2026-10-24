-- PicHub 初始化 Schema
-- 字符集 utf8mb4 / 排序 utf8mb4_general_ci / 引擎 InnoDB
-- 所有时间字段统一 UTC

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户组：配额与权限策略';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户账号';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邮箱令牌';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邮件发送日志';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='API访问令牌';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='S3兼容凭据';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='存储策略配置';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='相册';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='图片元数据';

CREATE TABLE `tags` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT               COMMENT '主键ID',
  `user_id`     BIGINT UNSIGNED NOT NULL                              COMMENT '用户ID',
  `name`        VARCHAR(32)     NOT NULL                              COMMENT '标签名称',
  `color`       VARCHAR(16)     NOT NULL DEFAULT ''                   COMMENT '标签颜色',
  `image_count` INT             NOT NULL DEFAULT 0                    COMMENT '关联图片数(冗余)',
  `created_at`  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_name` (`user_id`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='标签';

CREATE TABLE `image_tags` (
  `image_id`   BIGINT UNSIGNED NOT NULL                               COMMENT '图片ID',
  `tag_id`     BIGINT UNSIGNED NOT NULL                               COMMENT '标签ID',
  `created_at` DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3)  COMMENT '创建时间',
  PRIMARY KEY (`image_id`, `tag_id`),
  KEY `idx_tag` (`tag_id`, `image_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='图片-标签关联';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='分享链接';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='图片引用来源';

CREATE TABLE `image_referer_daily` (
  `image_id`     BIGINT UNSIGNED NOT NULL                             COMMENT '图片ID',
  `stat_date`    DATE            NOT NULL                             COMMENT '统计日期',
  `referer_host` VARCHAR(255)    NOT NULL                             COMMENT 'Referer域名',
  `hit_count`    BIGINT          NOT NULL DEFAULT 0                   COMMENT '当日次数',
  PRIMARY KEY (`image_id`, `stat_date`, `referer_host`),
  KEY `idx_date` (`stat_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='图片引用按天聚合';

CREATE TABLE `upload_counters` (
  `user_id`    BIGINT UNSIGNED NOT NULL                               COMMENT '用户ID',
  `stat_date`  DATE            NOT NULL                               COMMENT '统计日期(UTC)',
  `count`      INT             NOT NULL DEFAULT 0                     COMMENT '当日上传张数',
  `bytes`      BIGINT          NOT NULL DEFAULT 0                     COMMENT '当日上传字节数',
  `updated_at` DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`user_id`, `stat_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='每日上传用量';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='操作审计日志';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统全局配置';

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='注册邀请码';

-- ---------------------------- 初始化数据 ----------------------------
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
