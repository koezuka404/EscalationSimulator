package entity

import (
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

type Role string

const (
	RoleApplicant Role = "applicant"
	RoleAgent     Role = "agent"
	RoleAdmin     Role = "admin"
)

type AccountStatus string

const (
	StatusActive    AccountStatus = "active"
	StatusSuspended AccountStatus = "suspended"
	StatusDeleted   AccountStatus = "deleted"
)

var (
	ErrInvalidUserName = errors.New("名前は1文字以上、50文字以内で入力してください")
	ErrInvalidEmail    = errors.New("メールアドレスの形が違います。name@example.com のように入力してください")
	ErrInvalidPassword = errors.New("パスワードは8文字以上15文字以内にして、英字と数字をそれぞれ1文字以上入れてください")
	ErrEmailTaken      = errors.New("このメールアドレスはすでに使われています。別のメールアドレスを入力してください")
	ErrUserNotFound    = errors.New("利用者が見つかりませんでした")
	ErrLoginFailed     = errors.New("メールアドレスまたはパスワードが違います")
	ErrLoginLocked     = errors.New("ログインの失敗が続いたため、しばらくログインできません。時間をおいてもう一度試してください")
	ErrInvalidRefresh  = errors.New("ログインの期限が切れています。もう一度ログインしてください")
	ErrUnauthenticated = errors.New("ログインが必要です。もう一度ログインしてください")
	ErrForbidden       = errors.New("この操作は管理者だけができます")
	ErrNotApplicant    = errors.New("顧客を結びつけられるのは申請者だけです")
	ErrCustomerLink    = errors.New("結びつける顧客を選んでください")
)

type User struct {
	ID               string
	Email            string
	PasswordHash     string
	Name             string
	Role             Role
	CustomerID       string
	Status           AccountStatus
	AuthVersion      int
	FailedLoginCount int
	LockedUntil      time.Time
}

//会員登録用の申請者を作る所属顧客は空
func NewApplicant(name, email, password string) (User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 50 {
		return User{}, ErrInvalidUserName
	}
	if err := validateEmail(email); err != nil {
		return User{}, err
	}
	if err := validatePassword(password); err != nil {
		return User{}, err
	}
	return User{
		Email:       email,
		Name:        name,
		Role:        RoleApplicant,
		Status:      StatusActive,
		AuthVersion: 1,
	}, nil
}

func validateEmail(email string) error {
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || utf8.RuneCountInString(email) > 254 {
		return ErrInvalidEmail
	}
	return nil
}

func validatePassword(password string) error {
	count := utf8.RuneCountInString(password)
	if count < 8 || count > 15 {
		return ErrInvalidPassword
	}
	var letter, digit bool
	for _, r := range password {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			letter = true
		}
		if r >= '0' && r <= '9' {
			digit = true
		}
	}
	if !letter || !digit {
		return ErrInvalidPassword
	}
	return nil
}

//ログインを止めている時間内かを返す
func (u User) Locked(now time.Time) bool {
	return !u.LockedUntil.IsZero() && now.Before(u.LockedUntil)
}

//申請者を1つの顧客に結びつけ、変わったときは認証の版を上げる
func (u User) LinkCustomer(customerID string) (User, error) {
	if u.Role != RoleApplicant {
		return User{}, ErrNotApplicant
	}
	if strings.TrimSpace(customerID) == "" {
		return User{}, ErrCustomerLink
	}
	if u.CustomerID == customerID {
		return u, nil
	}
	u.CustomerID = customerID
	u.AuthVersion++
	return u, nil
}
