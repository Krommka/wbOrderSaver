package domain

import "errors"

type Payment struct {
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDT    int64  `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

func (p *Payment) Validate(orderUID string) error {
	if p.Transaction != orderUID {
		return errors.New("transaction должен совпадать с order_uid")
	}

	if len(p.Currency) != 3 {
		return errors.New("валюта должна быть 3 символа")
	}

	if p.Amount != p.DeliveryCost+p.GoodsTotal {
		return errors.New("amount должен равняться delivery_cost + goods_total")
	}

	return nil
}
