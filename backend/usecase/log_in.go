package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
	"escalator/usecase/crypto"
)

type LogIn struct {
	users       port.UserRepository
	sessions    port.SessionRepository
	secret      []byte
	issuer      string
	audience    string
	accessTTL   time.Duration
	refreshTTL  time.Duration
	maxFailures int
	lock        time.Duration
	audit       *AuditLog
}

type LoginResult struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
	CSRFToken    string
	User         entity.User
}

func NewLogIn(users port.UserRepository, sessions port.SessionRepository, secret []byte, issuer, audience string, accessTTL, refreshTTL time.Duration, maxFailures int, lock time.Duration, audit *AuditLog) *LogIn {
	return &LogIn{
		users:       users,
		sessions:    sessions,
		secret:      secret,
		issuer:      issuer,
		audience:    audience,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
		maxFailures: maxFailures,
		lock:        lock,
		audit:       audit,
	}
}

// メールアドレスとパスワードを確かめ、ログインする
func (l *LogIn) Execute(ctx context.Context, email, password string) (LoginResult, error) {
	now := time.Now()
	user, err := l.users.FindByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, rule.ErrUserNotFound) {
		return LoginResult{}, rule.ErrLoginFailed
	}
	if err != nil {
		return LoginResult{}, err
	}
	if rule.Locked(user, now) {
		return LoginResult{}, rule.ErrLoginLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, l.recordFailure(ctx, user, now)
	}
	if user.Status != entity.StatusActive {
		return LoginResult{}, rule.ErrLoginFailed
	}
	user.FailedLoginCount = 0
	user.LockedUntil = time.Time{}
	if err := l.users.UpdateLoginState(ctx, user); err != nil {
		return LoginResult{}, err
	}

	accessToken, expiresAt, err := crypto.IssueAccess(l.secret, l.issuer, l.audience, user.ID, user.AuthVersion, l.accessTTL, now)
	if err != nil {
		return LoginResult{}, err
	}
	refreshToken, refreshHash, err := crypto.NewSecretToken()
	if err != nil {
		return LoginResult{}, err
	}
	csrfToken, _, err := crypto.NewSecretToken()
	if err != nil {
		return LoginResult{}, err
	}
	if err := l.sessions.Save(ctx, entity.Session{
		UserID:    user.ID,
		TokenHash: refreshHash,
		FamilyID:  uuid.NewString(),
		ExpiresAt: now.Add(l.refreshTTL),
	}); err != nil {
		return LoginResult{}, err
	}
	l.audit.Record(ctx, user.ID, entity.AuditLogin, "user", user.ID, "ログインしました")
	return LoginResult{
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken,
		CSRFToken:    csrfToken,
		User:         user,
	}, nil
}

func (l *LogIn) recordFailure(ctx context.Context, user entity.User, now time.Time) error {
	user.FailedLoginCount++
	if user.FailedLoginCount >= l.maxFailures {
		user.FailedLoginCount = 0
		user.LockedUntil = now.Add(l.lock)
		if err := l.users.UpdateLoginState(ctx, user); err != nil {
			return err
		}
		l.audit.Record(ctx, user.ID, entity.AuditLoginLocked, "user", user.ID, "ログインの失敗が続いたため、ログインを止めました")
		return rule.ErrLoginLocked
	}
	if err := l.users.UpdateLoginState(ctx, user); err != nil {
		return err
	}
	return rule.ErrLoginFailed
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
