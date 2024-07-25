package gophermart

import (
	"sync/atomic"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func validateLogin(login string) error {
	if login == "" {
		return ErrLoginNotValid
	}
	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return ErrPasswordNotValid
	}
	return nil
}

func HashPassword(password string) (string, error) {
	cost := 14
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

type semaphore struct {
	lockedTime *atomic.Int64
}

func NewSemaphore() *semaphore {
	s := &semaphore{
		lockedTime: &atomic.Int64{},
	}
	s.lockedTime.Store(time.Now().UnixNano())

	return s
}

func (s *semaphore) Wait() {
	var delay int64 = 100
	for {
		if s.lockedTime.Load() < time.Now().UnixNano() {
			return
		}
		time.Sleep(time.Millisecond * time.Duration(delay))
	}
}

func (s *semaphore) Lock(d time.Duration) {
	s.lockedTime.Store(time.Now().Add(d).UnixNano())
}
