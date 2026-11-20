package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
	"escalator/usecase/crypto"
)

type ScreenConnection interface {
	CloseUser(userID string)
}

type Refresh struct {
	users      port.UserRepository
	sessions   port.SessionRepository
	secret     []byte
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
	screens    ScreenConnection
	audit      *AuditLog
}

func NewRefresh(users port.UserRepository, sessions port.SessionRepository, secret []byte, issuer, audience string, accessTTL, refreshTTL time.Duration, screens ScreenConnection, audit *AuditLog) *Refresh {
	return &Refresh{
		users:      users,
		sessions:   sessions,
		secret:     secret,
		issuer:     issuer,
		audience:   audience,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		screens:    screens,
		audit:      audit,
	}
}

// 再ログイン用の印を新しい印に替え、使い回しを見つけたら画面の接続を切る
func (r *Refresh) Execute(ctx context.Context, refreshToken string) (LoginResult, error) {
	if refreshToken == "" {
		return LoginResult{}, rule.ErrInvalidRefresh
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
		r.audit.Record(ctx, userID, entity.AuditRefreshReused, "user", userID, "無効な印の再利用を見つけました")
		if r.screens != nil {
			r.screens.CloseUser(userID)
			log.Printf("使い回された印を見つけたため、その人の画面の接続を切りました。user_id=%s", userID)
			r.audit.Record(ctx, userID, entity.AuditConnectionClosed, "user", userID, "画面の接続を切りました")
		}
		return LoginResult{}, rule.ErrInvalidRefresh
	}
	user, err := r.users.FindByID(ctx, userID)
	if err != nil || user.Status != entity.StatusActive {
		return LoginResult{}, rule.ErrInvalidRefresh
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
