package service

import (
	"context"
	"strings"
	"time"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/crypto"
	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
)

var allowedScopes = map[string]struct{}{
	"upload": {}, "read": {}, "write": {}, "delete": {}, "stats": {},
}

// TokenService 处理 API Token 与 S3 凭据。
type TokenService struct {
	cfg    *config.Config
	log    *logger.Logger
	repos  Repos
	cipher *crypto.Cipher
}

// NewTokenService 创建令牌服务。
func NewTokenService(d Deps) *TokenService {
	return &TokenService{cfg: d.Config, log: d.Log, repos: d.Repos, cipher: d.Cipher}
}

// ListTokens 返回用户的 API Token。
func (s *TokenService) ListTokens(ctx context.Context, userID uint64) ([]*dto.TokenResp, error) {
	list, err := s.repos.Token.ListByUser(ctx, userID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	out := make([]*dto.TokenResp, 0, len(list))
	for _, t := range list {
		out = append(out, dto.NewTokenResp(t))
	}
	return out, nil
}

// CreateToken 生成 API Token，明文仅返回一次。
func (s *TokenService) CreateToken(ctx context.Context, userID uint64, req *dto.TokenCreateReq) (*dto.TokenCreateResp, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 64 {
		return nil, errs.ErrInvalidParam.WithDetail("令牌名称长度需为 1-64 个字符")
	}
	scopes := normalizeScopes(req.Scopes)
	if len(scopes) == 0 {
		scopes = []string{"upload", "read", "write", "delete", "stats"}
	}
	if err := s.checkTokenQuota(ctx, userID); err != nil {
		return nil, err
	}
	plain, hash, prefix, err := auth.NewAPIToken()
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	t := &model.APIToken{
		UserID:      userID,
		Name:        name,
		TokenHash:   hash,
		TokenPrefix: prefix,
		Scopes:      strings.Join(scopes, ","),
		Status:      model.StatusEnabled,
	}
	if req.ExpiresInDays > 0 {
		if req.ExpiresInDays > 3650 {
			req.ExpiresInDays = 3650
		}
		t.ExpiresAt = ptrTime(model.Now().AddDate(0, 0, req.ExpiresInDays))
	}
	if err := s.repos.Token.Create(ctx, t); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return &dto.TokenCreateResp{Token: plain, Info: dto.NewTokenResp(t)}, nil
}

// DeleteToken 删除 API Token。
func (s *TokenService) DeleteToken(ctx context.Context, userID, id uint64) error {
	t, err := s.repos.Token.GetByID(ctx, id)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if t == nil || t.UserID != userID {
		return errs.ErrNotFound.WithMessage("令牌不存在")
	}
	if err := s.repos.Token.Delete(ctx, id); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	return nil
}

// SetTokenStatus 启用或禁用 API Token。
func (s *TokenService) SetTokenStatus(ctx context.Context, userID, id uint64, status int8) error {
	if status != model.StatusEnabled && status != model.StatusDisabled {
		return errs.ErrInvalidParam.WithDetail("状态值无效")
	}
	t, err := s.repos.Token.GetByID(ctx, id)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if t == nil || t.UserID != userID {
		return errs.ErrNotFound.WithMessage("令牌不存在")
	}
	if err := s.repos.Token.UpdateFields(ctx, id, map[string]any{"status": status}); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	return nil
}

// ValidateAPIToken 校验 API Token 并返回对应用户。
func (s *TokenService) ValidateAPIToken(ctx context.Context, plain, ip string) (*model.User, *model.APIToken, error) {
	plain = strings.TrimSpace(plain)
	if !strings.HasPrefix(plain, "ph_") {
		return nil, nil, errs.ErrUnauthorized
	}
	t, err := s.repos.Token.GetByHash(ctx, auth.HashToken(plain))
	if err != nil {
		return nil, nil, errs.ErrInternal.Wrap(err)
	}
	if t == nil {
		return nil, nil, errs.ErrUnauthorized.WithMessage("Token 无效")
	}
	if t.ExpiresAt != nil && t.ExpiresAt.Before(model.Now()) {
		return nil, nil, errs.ErrTokenExpired
	}
	u, err := s.repos.User.GetByID(ctx, t.UserID)
	if err != nil {
		return nil, nil, errs.ErrInternal.Wrap(err)
	}
	if u == nil || u.Status == model.UserStatusDisabled {
		return nil, nil, errs.ErrForbidden.WithMessage("账号不可用")
	}
	if err := s.repos.Token.IncrCallCount(ctx, t.ID, ip); err != nil && s.log != nil {
		s.log.Warn("incr token call count failed", logger.Err(err))
	}
	return u, t, nil
}

// ListS3Credentials 返回用户的 S3 凭据。
func (s *TokenService) ListS3Credentials(ctx context.Context, userID uint64) ([]*dto.S3CredResp, error) {
	list, err := s.repos.S3Cred.ListByUser(ctx, userID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	out := make([]*dto.S3CredResp, 0, len(list))
	for _, c := range list {
		out = append(out, dto.NewS3CredResp(c))
	}
	return out, nil
}

// CreateS3Credential 生成 S3 访问凭据，Secret 仅返回一次。
func (s *TokenService) CreateS3Credential(ctx context.Context, userID uint64, req *dto.S3CredCreateReq) (*dto.S3CredCreateResp, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 64 {
		return nil, errs.ErrInvalidParam.WithDetail("凭据名称长度需为 1-64 个字符")
	}
	if err := s.checkTokenQuota(ctx, userID); err != nil {
		return nil, err
	}
	ak, err := auth.NewAccessKey()
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	sk, err := auth.NewSecretKey()
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	enc, err := s.cipher.EncryptString(sk)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	c := &model.S3Credential{
		UserID:    userID,
		AccessKey: ak,
		SecretEnc: enc,
		Name:      name,
		Status:    model.StatusEnabled,
	}
	if err := s.repos.S3Cred.Create(ctx, c); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return &dto.S3CredCreateResp{AccessKey: ak, SecretKey: sk, Info: dto.NewS3CredResp(c)}, nil
}

// DeleteS3Credential 删除 S3 凭据。
func (s *TokenService) DeleteS3Credential(ctx context.Context, userID, id uint64) error {
	c, err := s.repos.S3Cred.GetByID(ctx, id)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if c == nil || c.UserID != userID {
		return errs.ErrNotFound.WithMessage("凭据不存在")
	}
	if err := s.repos.S3Cred.Delete(ctx, id); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	return nil
}

// ValidateS3Credential 校验 AccessKey 并返回明文 Secret 与用户。
func (s *TokenService) ValidateS3Credential(ctx context.Context, accessKey string) (*model.S3Credential, string, *model.User, error) {
	accessKey = strings.TrimSpace(accessKey)
	if accessKey == "" {
		return nil, "", nil, errs.ErrS3Signature
	}
	c, err := s.repos.S3Cred.GetByAccessKey(ctx, accessKey)
	if err != nil {
		return nil, "", nil, errs.ErrInternal.Wrap(err)
	}
	if c == nil {
		return nil, "", nil, errs.ErrS3Signature.WithMessage("AccessKey 不存在")
	}
	if c.Status != model.StatusEnabled {
		return nil, "", nil, errs.ErrS3KeyDisabled
	}
	secret, err := s.cipher.DecryptString(c.SecretEnc)
	if err != nil {
		return nil, "", nil, errs.ErrInternal.Wrap(err)
	}
	u, err := s.repos.User.GetByID(ctx, c.UserID)
	if err != nil {
		return nil, "", nil, errs.ErrInternal.Wrap(err)
	}
	if u == nil || u.Status == model.UserStatusDisabled {
		return nil, "", nil, errs.ErrS3KeyDisabled.WithMessage("账号不可用")
	}
	return c, secret, u, nil
}

// TouchS3Credential 记录 S3 凭据调用。
func (s *TokenService) TouchS3Credential(ctx context.Context, id uint64, ip string) {
	if err := s.repos.S3Cred.IncrCallCount(ctx, id, ip); err != nil && s.log != nil {
		s.log.Warn("incr s3 credential call count failed", logger.Err(err))
	}
}

func (s *TokenService) checkTokenQuota(ctx context.Context, userID uint64) error {
	if s.repos.Group == nil {
		return nil
	}
	u, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if u == nil || u.GroupID == 0 {
		return nil
	}
	g, err := s.repos.Group.GetByID(ctx, u.GroupID)
	if err != nil || g == nil || g.MaxTokens <= 0 {
		return nil
	}
	n, err := s.repos.Token.CountByUser(ctx, userID)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if n >= int64(g.MaxTokens) {
		return errs.ErrForbidden.WithMessage("令牌数量已达上限")
	}
	return nil
}

func normalizeScopes(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, raw := range in {
		scope := strings.ToLower(strings.TrimSpace(raw))
		if _, ok := allowedScopes[scope]; !ok {
			continue
		}
		if _, dup := seen[scope]; dup {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out
}

// ExpiryTime 返回 N 天后的过期时间，0 表示永不过期。
func ExpiryTime(days int) *time.Time {
	if days <= 0 {
		return nil
	}
	return ptrTime(model.Now().AddDate(0, 0, days))
}
