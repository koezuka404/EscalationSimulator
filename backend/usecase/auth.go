package usecase

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
	"escalator/scope"
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

type LogOut struct {
	secrets  port.SecretTokens
	sessions port.SessionRepository
	audit    *AuditLog
}

func NewLogOut(sessions port.SessionRepository, secrets port.SecretTokens, audit *AuditLog) *LogOut {
	return &LogOut{secrets: secrets, sessions: sessions, audit: audit}
}

//再ログイン用の印を無効にする
func (l *LogOut) Execute(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	hash := l.secrets.Hash(refreshToken)
	userID, err := l.sessions.FindUserIDByHash(ctx, hash)
	if err != nil && !errors.Is(err, rule.ErrInvalidRefresh) {
		return err
	}
	if err := l.sessions.RevokeByHash(ctx, hash); err != nil {
		return err
	}
	if userID != "" {
		l.audit.Record(ctx, userID, entity.AuditLogout, "user", userID, "ログアウトしました")
	}
	return nil
}

type ScreenConnection interface {
	CloseUser(userID string)
}

type Refresh struct {
	users    port.UserRepository
	sessions port.SessionRepository
	tokens   port.TokenIssuer
	secrets  port.SecretTokens
	policy   SessionPolicy
	screens  ScreenConnection
	audit    *AuditLog
}

func NewRefresh(users port.UserRepository, sessions port.SessionRepository, tokens port.TokenIssuer, secrets port.SecretTokens, policy SessionPolicy, screens ScreenConnection, audit *AuditLog) *Refresh {
	return &Refresh{
		users:    users,
		sessions: sessions,
		tokens:   tokens,
		secrets:  secrets,
		policy:   policy,
		screens:  screens,
		audit:    audit,
	}
}

//再ログイン用の印を新しい印に替え、使い回しを見つけたら画面の接続を切る
func (r *Refresh) Execute(ctx context.Context, refreshToken string) (LoginResult, error) {
	if refreshToken == "" {
		return LoginResult{}, rule.ErrInvalidRefresh
	}
	now := time.Now()
	raw, hash, err := r.secrets.New()
	if err != nil {
		return LoginResult{}, err
	}
	csrfToken, _, err := r.secrets.New()
	if err != nil {
		return LoginResult{}, err
	}
	userID, reused, err := r.sessions.Rotate(ctx, r.secrets.Hash(refreshToken), entity.Session{
		TokenHash: hash,
		ExpiresAt: now.Add(r.policy.RefreshTTL),
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
	accessToken, expiresAt, err := r.tokens.Issue(user.ID, user.AuthVersion, now)
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

type CurrentUser struct {
	users  port.UserRepository
	tokens port.TokenVerifier
}

func NewCurrentUser(users port.UserRepository, tokens port.TokenVerifier) *CurrentUser {
	return &CurrentUser{users: users, tokens: tokens}
}

//ログイン用トークンから、今の利用者を取る
func (c *CurrentUser) Execute(ctx context.Context, token string) (entity.User, error) {
	if token == "" {
		return entity.User{}, rule.ErrUnauthenticated
	}
	userID, authVersion, err := c.tokens.Parse(token)
	if err != nil || userID == "" {
		return entity.User{}, rule.ErrUnauthenticated
	}
	user, err := c.users.FindByID(ctx, userID)
	if err != nil || user.Status != entity.StatusActive || user.AuthVersion != authVersion {
		return entity.User{}, rule.ErrUnauthenticated
	}
	return user, nil
}

type SignUp struct {
	users    port.UserRepository
	password port.PasswordHasher
}

func NewSignUp(users port.UserRepository, password port.PasswordHasher) *SignUp {
	return &SignUp{users: users, password: password}
}

//申請者を登録する
func (s *SignUp) Execute(ctx context.Context, name, email, password string) (entity.User, error) {
	user, err := rule.NewApplicant(name, email, password)
	if err != nil {
		return entity.User{}, err
	}
	hash, err := s.password.Hash(password)
	if err != nil {
		return entity.User{}, err
	}
	user.PasswordHash = hash
	return s.users.Save(ctx, user)
}

type SessionPolicy struct {
	RefreshTTL  time.Duration
	MaxFailures int
	Lock        time.Duration
}
