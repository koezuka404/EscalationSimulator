package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/repository"
	"escalator/usecase/crypto"
)

type ScreenConnection interface {
	CloseUser(userID string)
}

type Refresh struct {
	users      repository.UserRepository
	sessions   repository.SessionRepository
	secret     []byte
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
	screens    ScreenConnection
}

func NewRefresh(users repository.UserRepository, sessions repository.SessionRepository, secret []byte, issuer, audience string, accessTTL, refreshTTL time.Duration, screens ScreenConnection) *Refresh {
	return &Refresh{
		users:      users,
		sessions:   sessions,
		secret:     secret,
		issuer:     issuer,
		audience:   audience,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		screens:    screens,
	}
}

//再ログイン用の印を新しい印に替え、使い回しを見つけたら画面の接続を切る
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
		if r.screens != nil {
			r.screens.CloseUser(userID)
		}
		log.Printf("使い回された印を見つけたため、その人の画面の接続を切りました。user_id=%s", userID)
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
