package application

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(hash), err
}

func comparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
