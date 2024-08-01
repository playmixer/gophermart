package gophermart

import (
	"fmt"
	"sync"
	"time"

	"errors"

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

type circuitBreakerState int

const (
	cbOpen circuitBreakerState = iota
	cbClose
	cbHalfOpen
)

type circuitBreaker struct {
	mu          *sync.Mutex
	expireDelay int64
	state       circuitBreakerState
}

func newCircuitBreaker() *circuitBreaker {
	cb := &circuitBreaker{
		mu:          &sync.Mutex{},
		expireDelay: 0,
		state:       cbClose,
	}

	return cb
}

func (cb *circuitBreaker) execute(request func() (int64, error)) error {
	cb.mu.Lock()
	switch cb.state {
	case cbOpen:
		current := time.Now().Unix()
		if current > cb.expireDelay {
			cb.state = cbHalfOpen
		} else {
			cb.mu.Unlock()
			return errors.New("service unavailabale")
		}
	case cbHalfOpen:
		cb.state = cbOpen
	default:
	}
	cb.mu.Unlock()

	delay, err := request()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	timeDelay := time.Duration(delay) * time.Second
	if timeDelay > 0 {
		cb.expireDelay = time.Now().Add(timeDelay).Unix()
	}
	if err != nil {
		cb.state = cbOpen
		time.Sleep(timeDelay)
		return fmt.Errorf("request error: %w", err)
	}

	if timeDelay > 0 {
		cb.state = cbOpen
		time.Sleep(timeDelay)
		return nil
	}

	cb.state = cbClose
	return nil
}
