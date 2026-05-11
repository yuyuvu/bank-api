package security

import "golang.org/x/crypto/bcrypt"

// HashPassword хеширует пароль пользователя через bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// VerifyPassword сравнивает сохранённый bcrypt-хеш и пароль из запроса
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// HashCVV хеширует CVV так же, как и пароль
func HashCVV(cvv string) (string, error) {
	return HashPassword(cvv)
}
