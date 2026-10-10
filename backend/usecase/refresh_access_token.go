package usecase

import (
	"context"
	"time"

	"escalator/entity"
	"escalator/repository"
	"escalator/usecase/crypto"
)

type Refresh struct {
	users      repository.UserRepository
	sessions   repository.SessionRepository
	secret     []byte
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewRefresh(users repository.UserRepository, sessions repository.SessionRepository, secret []byte, issuer, audience string, accessTTL, refreshTTL time.Duration) *Refresh {
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

//再ログイン用の印を新しい印に替え、ログイン用トークンを出し直す
func (r *Refresh) Execute(ctx context.Context, refreshToken string) (LoginResult, error) {
	if refreshToken == "" {
		return LoginResult{}, entity.ErrInvalidRefresh
	}
	now := time.Now()
	raw, hash, err := crypto.NewSecretToken()
	if err != nil {
		return LoginResult{}, err
	}
	csrfToken, _, err := crypto.NewSecretToken()
	if err != nil {
		return LoginResult{}, err
	}
	userID, reused, err := r.sessions.Rotate(ctx, crypto.Hash(refreshToken), entity.Session{
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
		return LoginResult{}, entity.ErrInvalidRefresh
	}
	user, err := r.users.FindByID(ctx, userID)
	if err != nil || user.Status != entity.StatusActive {
		return LoginResult{}, entity.ErrInvalidRefresh
	}
	accessToken, expiresAt, err := crypto.IssueAccess(r.secret, r.issuer, r.audience, user.ID, user.AuthVersion, r.accessTTL, now)
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
