package crypto

import "golang.org/x/crypto/bcrypt"

type Bcrypt struct {
	cost int
}

func NewBcrypt(cost int) *Bcrypt {
	return &Bcrypt{cost: cost}
}

//パスワードをハッシュにする
func (b *Bcrypt) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

//パスワードがハッシュと一致するかを返す
func (b *Bcrypt) Matches(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
