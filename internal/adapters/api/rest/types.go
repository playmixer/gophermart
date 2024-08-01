package rest

import (
	"time"

	"github.com/playmixer/gophermart/internal/adapters/store/model"
)

type tRegistration struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type tAuthorization struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type tOrderByUser struct {
	uploadedAt time.Time
	Accrual    *float32          `json:"accrual,omitempty"`
	Number     string            `json:"number"`
	UploadedAt string            `json:"uploaded_at"`
	Status     model.OrderStatus `json:"status"`
}

func (o *tOrderByUser) Prepare() *tOrderByUser {
	o.UploadedAt = o.uploadedAt.Format(time.RFC3339)
	return o
}

type tBalanceByUser struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

type tWithdraw struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

type tWithdrawBalance struct {
	processedAt time.Time
	Order       string  `json:"order"`
	ProcessedAt string  `json:"processed_at"`
	Sum         float32 `json:"sum"`
}

func (w *tWithdrawBalance) Prepare() *tWithdrawBalance {
	w.ProcessedAt = w.processedAt.Format(time.RFC3339)
	return w
}
