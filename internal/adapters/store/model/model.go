package model

import (
	"time"
)

type User struct {
	CreatedAt    time.Time `gorm:"type:time"`
	UpdatedAt    time.Time `gorm:"type:time"`
	Login        string    `gorm:"unique"`
	PasswordHash string    `gorm:"type:string"`
	ID           uint      `gorm:"primarykey"`
}

type OrderStatus string

const (
	OrderStateNew        OrderStatus = "NEW"
	OrderStateProcessing OrderStatus = "PROCESSING"
	OrderStateInvalid    OrderStatus = "INVALID"
	OrderStateProcessed  OrderStatus = "PROCESSED"
)

type Order struct {
	CreatedAt time.Time   `gorm:"type:time"`
	UpdatedAt time.Time   `gorm:"type:time"`
	Number    string      `gorm:"unique,index"`
	Status    OrderStatus `gorm:"default:NEW"`
	User      User
	ID        uint    `gorm:"primarykey"`
	UserID    uint    `gorm:"index"`
	Accrual   float32 `gorm:"type:float"`
}

type Balance struct {
	CreatedAt time.Time `gorm:"type:time"`
	UpdatedAt time.Time `gorm:"type:time"`
	User      User
	ID        uint    `gorm:"primarykey"`
	UserID    uint    `gorm:"unique"`
	Current   float32 `gorm:"type:float"`
	Withdrawn float32 `gorm:"type:float"`
}

type WithdrawBalance struct {
	CreatedAt  time.Time `gorm:"type:time"`
	UpdatedAt  time.Time `gorm:"type:time"`
	OderNumber string    `gorm:"type:string"`
	Balance    Balance
	ID         uint    `gorm:"primarykey"`
	BalanceID  uint    `gorm:"index"`
	Sum        float32 `gorm:"type:float"`
}
