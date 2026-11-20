package rule

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"escalator/entity"
)

//会員登録用の申請者を作る所属顧客は空
func NewApplicant(name, email, password string) (entity.User, error) {
	return newUser(name, email, password, entity.RoleApplicant)
}

//担当者を作る所属顧客は空
func NewAgent(name, email, password string) (entity.User, error) {
	return newUser(name, email, password, entity.RoleAgent)
}

//管理者を作る所属顧客は空
func NewAdmin(name, email, password string) (entity.User, error) {
	return newUser(name, email, password, entity.RoleAdmin)
}

func newUser(name, email, password string, role entity.Role) (entity.User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 50 {
		return entity.User{}, ErrInvalidUserName
	}
	if err := validateEmail(email); err != nil {
		return entity.User{}, err
	}
	if err := validatePassword(password); err != nil {
		return entity.User{}, err
	}
	return entity.User{
		Email:       email,
		Name:        name,
		Role:        role,
		Status:      entity.StatusActive,
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
func Locked(user entity.User, now time.Time) bool {
	return !user.LockedUntil.IsZero() && now.Before(user.LockedUntil)
}

//申請者を1つの顧客に結びつけ、変わったときは認証の版を上げる
func LinkCustomer(user entity.User, customerID string) (entity.User, error) {
	if user.Role != entity.RoleApplicant {
		return entity.User{}, ErrNotApplicant
	}
	if strings.TrimSpace(customerID) == "" {
		return entity.User{}, ErrCustomerLink
	}
	if user.CustomerID == customerID {
		return user, nil
	}
	user.CustomerID = customerID
	user.AuthVersion++
	return user, nil
}

//利用者を停止し、認証の版を上げる
func Suspend(user entity.User) (entity.User, error) {
	if user.Status == entity.StatusDeleted {
		return entity.User{}, ErrUserNotFound
	}
	if user.Status == entity.StatusSuspended {
		return entity.User{}, ErrAlreadySuspended
	}
	user.Status = entity.StatusSuspended
	user.AuthVersion++
	return user, nil
}

//停止中の利用者を利用中に戻し、認証の版を上げる
func Resume(user entity.User) (entity.User, error) {
	if user.Status == entity.StatusDeleted {
		return entity.User{}, ErrUserNotFound
	}
	if user.Status == entity.StatusActive {
		return entity.User{}, ErrAlreadyActive
	}
	if user.Status != entity.StatusSuspended {
		return entity.User{}, ErrUserNotFound
	}
	user.Status = entity.StatusActive
	user.AuthVersion++
	return user, nil
}

//停止中の利用者を削除し、メールアドレスを匿名化して認証の版を上げる
func Delete(user entity.User) (entity.User, error) {
	if user.Status == entity.StatusDeleted || user.ID == "" {
		return entity.User{}, ErrUserNotFound
	}
	if user.Status != entity.StatusSuspended {
		return entity.User{}, ErrDeleteOnlySuspended
	}
	user.Status = entity.StatusDeleted
	user.Email = "deleted." + user.ID + "@deleted.invalid"
	user.AuthVersion++
	return user, nil
}

//役割を変え、申請者には顧客を必須にし、それ以外は所属を空にする
func ChangeRole(user entity.User, role entity.Role, customerID string) (entity.User, error) {
	if user.Status == entity.StatusDeleted {
		return entity.User{}, ErrUserNotFound
	}
	switch role {
	case entity.RoleApplicant, entity.RoleAgent, entity.RoleAdmin:
	default:
		return entity.User{}, ErrInvalidRole
	}
	if user.Role == role {
		return entity.User{}, ErrRoleSame
	}
	customerID = strings.TrimSpace(customerID)
	if role == entity.RoleApplicant {
		if customerID == "" {
			return entity.User{}, ErrApplicantNeedsCustomer
		}
	} else {
		customerID = ""
	}
	user.Role = role
	user.CustomerID = customerID
	user.AuthVersion++
	return user, nil
}
