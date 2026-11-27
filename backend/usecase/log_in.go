package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
	"escalator/scope"
	"github.com/google/uuid"
)

type LogIn struct {
	users    port.UserRepository
	attempts port.LoginAttemptRepository
	sessions port.SessionRepository
	password port.PasswordHasher
	tokens   port.TokenIssuer
	secrets  port.SecretTokens
	policy   SessionPolicy
	audit    *AuditLog
}

type LoginResult struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
	CSRFToken    string
	User         entity.User
}

func NewLogIn(users port.UserRepository, attempts port.LoginAttemptRepository, sessions port.SessionRepository, password port.PasswordHasher, tokens port.TokenIssuer, secrets port.SecretTokens, policy SessionPolicy, audit *AuditLog) *LogIn {
	return &LogIn{
		users:    users,
		attempts: attempts,
		sessions: sessions,
		password: password,
		tokens:   tokens,
		secrets:  secrets,
		policy:   policy,
		audit:    audit,
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
	if !l.password.Matches(user.PasswordHash, password) {
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

	accessToken, expiresAt, err := l.tokens.Issue(user.ID, user.AuthVersion, now)
	if err != nil {
		return LoginResult{}, err
	}
	refreshToken, refreshHash, err := l.secrets.New()
	if err != nil {
		return LoginResult{}, err
	}
	csrfToken, _, err := l.secrets.New()
	if err != nil {
		return LoginResult{}, err
	}
	if err := l.sessions.Save(ctx, entity.Session{
		UserID:    user.ID,
		TokenHash: refreshHash,
		FamilyID:  uuid.NewString(),
		ExpiresAt: now.Add(l.policy.RefreshTTL),
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
	if user.FailedLoginCount >= l.policy.MaxFailures {
		user.FailedLoginCount = 0
		user.LockedUntil = now.Add(l.policy.Lock)
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
	if attempt.FailedCount >= l.policy.MaxFailures {
		attempt.FailedCount = 0
		attempt.LockedUntil = now.Add(l.policy.Lock)
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
