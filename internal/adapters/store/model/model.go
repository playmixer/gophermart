package model

import "time"

type User struct {
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Login        string `gorm:"unique"`
	PasswordHash string
	ID           uint `gorm:"primarykey"`
}

type OrderStatus string

const (
	OrderStateNew        OrderStatus = "NEW"
	OrderStateProcessing OrderStatus = "PROCESSING"
	OrderStateInvalid    OrderStatus = "INVALID"
	OrderStateProcessed  OrderStatus = "PROCESSED"
)

type Order struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	Number    string      `gorm:"unique"`
	Status    OrderStatus `gorm:"default:NEW"`
	User      User
	ID        uint `gorm:"primarykey"`
	UserID    uint `gorm:"index"`
	Accrual   float32
}

type Balance struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	User      User
	ID        uint `gorm:"primarykey"`
	UserID    uint `gorm:"unique"`
	Current   float32
	Withdrawn float32
}

type WithdrawBalance struct {
	CreatedAt  time.Time
	UpdatedAt  time.Time
	OderNumber string
	Balance    Balance
	ID         uint `gorm:"primarykey"`
	BalanceID  uint `gorm:"index"`
	Sum        float32
}
