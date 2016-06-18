package auth

import (
	"crypto/rand"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"io"
	"regexp"
)

const PasswordCost = 10 // for bcrypt

func CheckPassword(pw string) error {
	// disallow whitespace
	r, err := regexp.Compile("\\s")
	if err != nil {
		return err
	}
	if r.MatchString(pw) {
		return errors.New("password cannot contain whitespace")
	}
	// minimum length 8 chars
	if len(pw) < 8 {
		return errors.New("password must be at least 8 chars")
	}

	return nil
}

func CheckUsername(usr string) error {
	if usr == "guest" {
		return errors.New("username guest already taken")
	}

	// regex verification
	r, err := regexp.Compile("^[a-z]{1}([_]{0,1}[a-zA-Z0-9]{1,})+$")
	if err != nil {
		return err
	}
	if r.MatchString(usr) {
		return nil
	}
	msg := "username isn't valid."
	return errors.New(msg)
}

func ValidatePassword(pwh, pw []byte) bool {
	err := bcrypt.CompareHashAndPassword(pwh, pw)
	if err != nil {
		return false
	}
	return true
}

func PasswordEncrypt(pw string) ([]byte, error) {
	pw_bytes := []byte(pw)
	pw_encrypt, err := bcrypt.GenerateFromPassword(pw_bytes, PasswordCost)
	if err != nil {
		return nil, err
	}
	return pw_encrypt, nil
}

// Taken from 'gorilla toolkit secure cookie'
func GenerateRandomKey(strength int) []byte {
	buffer := make([]byte, strength)
	if _, err := io.ReadFull(rand.Reader, buffer); err != nil {
		return nil
	}
	return buffer
}
