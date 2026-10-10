package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"escalator/domain"
	"escalator/infra/token"
)

type LogIn struct {
	users       domain.UserRepository
	sessions    domain.SessionRepository
	secret      []byte
	issuer      string
	audience    string
	accessTTL   time.Duration
	refreshTTL  time.Duration
	maxFailures int
	lock        time.Duration
}

type LoginResult struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
	CSRFToken    string
	User         domain.User
}

func NewLogIn(users domain.UserRepository, sessions domain.SessionRepository, secret []byte, issuer, audience string, accessTTL, refreshTTL time.Duration, maxFailures int, lock time.Duration) *LogIn {
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
	}
}

//メールアドレスとパスワードを確かめ、ログインする
func (l *LogIn) Execute(ctx context.Context, email, password string) (LoginResult, error) {
	now := time.Now()
	user, err := l.users.FindByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, domain.ErrUserNotFound) {
		return LoginResult{}, domain.ErrLoginFailed
	}
	if err != nil {
		return LoginResult{}, err
	}
	if user.Locked(now) {
		return LoginResult{}, domain.ErrLoginLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, l.recordFailure(ctx, user, now)
	}
	if user.Status != domain.StatusActive {
		return LoginResult{}, domain.ErrLoginFailed
	}
	user.FailedLoginCount = 0
	user.LockedUntil = time.Time{}
	if err := l.users.UpdateLoginState(ctx, user); err != nil {
		return LoginResult{}, err
	}

	accessToken, expiresAt, err := token.IssueAccess(l.secret, l.issuer, l.audience, user.ID, user.AuthVersion, l.accessTTL, now)
	if err != nil {
		return LoginResult{}, err
	}
	refreshToken, refreshHash, err := token.NewSecretToken()
	if err != nil {
		return LoginResult{}, err
	}
	csrfToken, _, err := token.NewSecretToken()
	if err != nil {
		return LoginResult{}, err
	}
	if err := l.sessions.Save(ctx, domain.Session{
		UserID:    user.ID,
		TokenHash: refreshHash,
		FamilyID:  uuid.NewString(),
		ExpiresAt: now.Add(l.refreshTTL),
	}); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken,
		CSRFToken:    csrfToken,
		User:         user,
	}, nil
}

func (l *LogIn) recordFailure(ctx context.Context, user domain.User, now time.Time) error {
	user.FailedLoginCount++
	if user.FailedLoginCount >= l.maxFailures {
		user.FailedLoginCount = 0
		user.LockedUntil = now.Add(l.lock)
		if err := l.users.UpdateLoginState(ctx, user); err != nil {
			return err
		}
		return domain.ErrLoginLocked
	}
	if err := l.users.UpdateLoginState(ctx, user); err != nil {
		return err
	}
	return domain.ErrLoginFailed
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
