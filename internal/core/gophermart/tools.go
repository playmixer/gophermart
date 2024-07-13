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
	locked *atomic.Bool
}

func NewSemaphore() *semaphore {
	s := &semaphore{
		locked: &atomic.Bool{},
	}
	s.locked.Store(false)

	return s
}

func (s *semaphore) Wait() {
	var delay int64 = 100
	for {
		if !s.locked.Load() {
			return
		}
		time.Sleep(time.Millisecond * time.Duration(delay))
	}
}

func (s *semaphore) Lock(d time.Duration) {
	s.locked.Store(true)
	go s.unlock(d)
}

func (s *semaphore) unlock(d time.Duration) {
	time.Sleep(d)
	s.locked.Store(false)
}
