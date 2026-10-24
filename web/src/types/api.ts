// 与 internal/dto 保持一致的前端类型定义。

export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface PageResp<T> {
  list: T[]
  total: number
  cursor?: number
  size?: number
}

export interface SiteConfig {
  name: string
  description: string
  logo_url: string
  icp: string
  site_url: string
  register_enabled: boolean
  register_mode: string
  email_verify_level: 'off' | 'soft' | 'force'
  captcha_enabled: boolean
  default_visibility: 'public' | 'private'
  max_file_size: number
  allowed_extensions: string[]
  thumbnail_enabled: boolean
  thumbnail_sizes: string[]
  s3_gateway_enabled: boolean
  s3_gateway_prefix: string
}

export interface User {
  id: number
  username: string
  email: string
  nickname: string
  avatar_url: string
  role: number
  status: number
  email_verified: boolean
  group_id: number
  quota_bytes: number
  used_bytes: number
  max_file_size: number
  daily_limit: number
  default_visibility: 'public' | 'private'
  storage_config_id: number | null
  referer_stat_enabled: boolean
  last_login_at: string | null
  created_at: string
}

export interface Captcha {
  captcha_id: string
  image_base64: string
  expires_in: number
}

export interface AuthResult {
  access_token: string
  refresh_token: string
  expires_in: number
  user: User
}

export interface CheckResult {
  available: boolean
  message: string
}

export interface HostStat {
  host: string
  hits: number
  images: number
}

export interface DayStat {
  date: string
  hits: number
}

export interface UserStats {
  image_count: number
  album_count: number
  share_count: number
  trash_count: number
  storage_bytes: number
  used_bytes: number
  quota_bytes: number
  daily_limit: number
  today_uploads: number
  referer_hits: number
  top_hosts: HostStat[]
  trend: DayStat[]
}

export interface Tag {
  id: number
  name: string
  color: string
  image_count: number
  created_at: string
}

export interface Album {
  id: number
  name: string
  description: string
  cover_image_id: number | null
  cover_url: string
  is_public: boolean
  sort_order: number
  image_count: number
  created_at: string
  updated_at: string
}

export interface Image {
  id: number
  hash: string
  name: string
  origin_name: string
  ext: string
  mime: string
  size: number
  width: number
  height: number
  url: string
  thumb_url: string
  is_public: boolean
  album_id: number | null
  tags?: Tag[]
  ref_count: number
  status: number
  created_at: string
  deleted_at: string | null
}

export interface ImageListQuery {
  keyword?: string
  album_id?: number
  tag_id?: number
  ext?: string
  referred?: boolean
  is_public?: boolean
  cursor?: number
  page?: number
  size?: number
  trash?: boolean
}

export interface ImageUpdatePayload {
  display_name?: string
  album_id?: number
  clear_album?: boolean
  tags?: string[]
  is_public?: boolean
}

export interface BatchResult {
  affected: number
}

export interface Referer {
  id: number
  host: string
  url: string
  hits: number
  first_seen_at: string
  last_seen_at: string
}

export interface RefererSummary {
  total_hits: number
  host_count: number
}

export interface UploadResult {
  id: number
  url: string
  thumb_url: string
  markdown: string
  html: string
  bbcode: string
  size: number
  width: number
  height: number
  hash: string
  dedup: boolean
}

export interface UploadCheckResult {
  exists: boolean
  image?: UploadResult
}

export interface UploadOptions {
  album_id?: number
  tags?: string
  is_public?: boolean
  storage_id?: number
}

export interface ChunkInitResult {
  upload_id: string
  uploaded: number[]
  chunk_size: number
}

export interface Token {
  id: number
  name: string
  prefix: string
  scopes: string[]
  expires_at: string | null
  last_used_at: string | null
  call_count: number
  status: number
  created_at: string
}

export interface TokenCreateResult {
  token: string
  info: Token
}

export interface S3Credential {
  id: number
  name: string
  access_key: string
  call_count: number
  last_used_at: string | null
  status: number
  created_at: string
}

export interface S3CredentialCreateResult {
  access_key: string
  secret_key: string
  info: S3Credential
}

export interface Share {
  id: number
  slug: string
  target_type: string
  target_id: number
  has_password: boolean
  expires_at: string | null
  max_views: number
  views: number
  status: number
  url: string
  created_at: string
}

export interface ShareAccess {
  share: Share
  album?: Album
  images?: Image[]
}

export interface StorageStrategy {
  id: number
  user_id: number | null
  name: string
  driver: string
  domain: string
  path_prefix: string
  is_default: boolean
  status: number
  root_path?: string
  path_template?: string
  endpoint?: string
  region?: string
  bucket?: string
  path_style: boolean
  use_ssl: boolean | null
  public_url?: string
  url?: string
  has_secret: boolean
  last_test_at: string | null
  last_test_ok: boolean | null
  created_at: string
  updated_at: string
}

export interface StorageTestResult {
  ok: boolean
  message: string
}

export interface AdminUser {
  id: number
  username: string
  email: string
  nickname: string
  avatar_url: string
  role: number
  status: number
  email_verified: boolean
  group_id: number
  quota_bytes: number
  used_bytes: number
  max_file_size: number
  daily_limit: number
  referer_stat_enabled: boolean
  register_ip: string
  last_login_ip: string
  last_login_at: string | null
  created_at: string
}

export interface Group {
  id: number
  name: string
  description: string
  quota_bytes: number
  max_file_size: number
  daily_limit: number
  max_albums: number
  max_tokens: number
  max_shares: number
  allow_register: boolean
  allow_custom_storage: boolean
  is_default: boolean
  status: number
  user_count: number
  created_at: string
  updated_at: string
}

export interface SettingItem {
  key: string
  value: string
  type: string
  is_secret: boolean
  description: string
  updated_at: string
}

export interface OperationLog {
  id: number
  user_id: number
  action: string
  target_type: string
  target_id: number
  ip: string
  user_agent: string
  detail?: unknown
  created_at: string
}

export interface EmailLog {
  id: number
  user_id: number
  to_email: string
  subject: string
  template: string
  status: number
  error: string
  retry: number
  sent_at: string | null
  created_at: string
}

export interface Invite {
  id: number
  code: string
  creator_id: number
  max_uses: number
  used_count: number
  group_id: number
  expires_at: string | null
  status: number
  created_at: string
}

export interface Dashboard {
  user_count: number
  image_count: number
  storage_bytes: number
  today_uploads: number
  referer_hits: number
  album_count: number
  share_count: number
  trash_count: number
  group_count: number
  invite_count: number
  queue_ready: number
  queue_delayed: number
  queue_dead: number
}
