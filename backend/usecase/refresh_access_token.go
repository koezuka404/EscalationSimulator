package usecase

import (
	"context"
	"time"

	"escalator/domain"
	"escalator/infra/token"
)

type Refresh struct {
	users      domain.UserRepository
	sessions   domain.SessionRepository
	secret     []byte
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewRefresh(users domain.UserRepository, sessions domain.SessionRepository, secret []byte, issuer, audience string, accessTTL, refreshTTL time.Duration) *Refresh {
	return &Refresh{
		users:      users,
		sessions:   sessions,
		secret:     secret,
		issuer:     issuer,
		audience:   audience,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// 再ログイン用の印を新しい印に替え、ログイン用トークンを出し直す。
func (r *Refresh) Execute(ctx context.Context, refreshToken string) (LoginResult, error) {
	if refreshToken == "" {
		return LoginResult{}, domain.ErrInvalidRefresh
	}
	now := time.Now()
	raw, hash, err := token.NewSecretToken()
	if err != nil {
		return LoginResult{}, err
	}
	csrfToken, _, err := token.NewSecretToken()
	if err != nil {
		return LoginResult{}, err
	}
	userID, reused, err := r.sessions.Rotate(ctx, token.Hash(refreshToken), domain.Session{
		TokenHash: hash,
		ExpiresAt: now.Add(r.refreshTTL),
	}, now)
	if err != nil {
		return LoginResult{}, err
	}
	if reused {
		if err := r.users.BumpAuthVersion(ctx, userID); err != nil {
			return LoginResult{}, err
		}
		return LoginResult{}, domain.ErrInvalidRefresh
	}
	user, err := r.users.FindByID(ctx, userID)
	if err != nil || user.Status != domain.StatusActive {
		return LoginResult{}, domain.ErrInvalidRefresh
	}
	accessToken, expiresAt, err := token.IssueAccess(r.secret, r.issuer, r.audience, user.ID, user.AuthVersion, r.accessTTL, now)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		RefreshToken: raw,
		CSRFToken:    csrfToken,
		User:         user,
	}, nil
}
