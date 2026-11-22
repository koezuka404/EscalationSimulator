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
	"escalator/scope"
	"escalator/usecase/crypto"
)

type LogIn struct {
	users       port.UserRepository
	attempts    port.LoginAttemptRepository
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

func NewLogIn(users port.UserRepository, attempts port.LoginAttemptRepository, sessions port.SessionRepository, secret []byte, issuer, audience string, accessTTL, refreshTTL time.Duration, maxFailures int, lock time.Duration, audit *AuditLog) *LogIn {
	return &LogIn{
		users:       users,
		attempts:    attempts,
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

//メールアドレスとパスワードを確かめ、ログインする
func (l *LogIn) Execute(ctx context.Context, email, password string) (LoginResult, error) {
	now := time.Now()
	email = normalizeEmail(email)
	ip := scope.ClientIP(ctx)
	if err := l.rejectStoppedAttempt(ctx, email, ip, now); err != nil {
		return LoginResult{}, err
	}
	user, err := l.users.FindByEmail(ctx, email)
	if errors.Is(err, rule.ErrUserNotFound) {
		return LoginResult{}, l.recordUnknownFailure(ctx, email, ip, now)
	}
	if err != nil {
		return LoginResult{}, err
	}
	if rule.Locked(user, now) {
		return LoginResult{}, rule.ErrLoginLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, l.recordFailure(ctx, user, email, ip, now)
	}
	if user.Status != entity.StatusActive {
		return LoginResult{}, rule.ErrLoginFailed
	}
	user.FailedLoginCount = 0
	user.LockedUntil = time.Time{}
	if err := l.users.UpdateLoginState(ctx, user); err != nil {
		return LoginResult{}, err
	}
	if err := l.clearAttempt(ctx, email, ip, now); err != nil {
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

func (l *LogIn) recordFailure(ctx context.Context, user entity.User, email, ip string, now time.Time) error {
	user.FailedLoginCount++
	stopped := false
	if user.FailedLoginCount >= l.maxFailures {
		user.FailedLoginCount = 0
		user.LockedUntil = now.Add(l.lock)
		stopped = true
	}
	if err := l.users.UpdateLoginState(ctx, user); err != nil {
		return err
	}
	if stopped {
		l.audit.Record(ctx, user.ID, entity.AuditLoginLocked, "user", user.ID, "ログインの失敗が続いたため、ログインを止めました")
	}
	attemptStopped, err := l.noteFailure(ctx, email, ip, now)
	if err != nil {
		return err
	}
	if stopped || attemptStopped {
		return rule.ErrLoginLocked
	}
	return rule.ErrLoginFailed
}

func (l *LogIn) recordUnknownFailure(ctx context.Context, email, ip string, now time.Time) error {
	stopped, err := l.noteFailure(ctx, email, ip, now)
	if err != nil {
		return err
	}
	if stopped {
		return rule.ErrLoginLocked
	}
	return rule.ErrLoginFailed
}

func (l *LogIn) rejectStoppedAttempt(ctx context.Context, email, ip string, now time.Time) error {
	attempt, err := l.attempts.Find(ctx, email, ip)
	if err != nil {
		return err
	}
	if rule.AttemptLocked(attempt, now) {
		return rule.ErrLoginLocked
	}
	return nil
}

//メールと接続元の失敗回数を1つ増やす
func (l *LogIn) noteFailure(ctx context.Context, email, ip string, now time.Time) (bool, error) {
	attempt, err := l.attempts.Find(ctx, email, ip)
	if err != nil {
		return false, err
	}
	attempt.Email = email
	attempt.IPAddress = ip
	attempt.FailedCount++
	attempt.LastFailedAt = now
	attempt.UpdatedAt = now
	stopped := false
	if attempt.FailedCount >= l.maxFailures {
		attempt.FailedCount = 0
		attempt.LockedUntil = now.Add(l.lock)
		stopped = true
	}
	if err := l.attempts.Save(ctx, attempt); err != nil {
		return false, err
	}
	return stopped, nil
}

//ログインできたメールと接続元の失敗回数を戻す
func (l *LogIn) clearAttempt(ctx context.Context, email, ip string, now time.Time) error {
	attempt, err := l.attempts.Find(ctx, email, ip)
	if err != nil {
		return err
	}
	if attempt.ID == "" {
		return nil
	}
	attempt.FailedCount = 0
	attempt.LockedUntil = time.Time{}
	attempt.UpdatedAt = now
	return l.attempts.Save(ctx, attempt)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
