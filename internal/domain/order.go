package domain

import (
	"errors"
	"regexp"
	"time"
)

type Order struct {
	OrderUID          string    `json:"order_uid"`
	TrackNumber       string    `json:"track_number"`
	Entry             string    `json:"entry"`
	Delivery          Delivery  `json:"delivery"`
	Payment           Payment   `json:"payment"`
	Items             []Item    `json:"items"`
	Locale            string    `json:"locale"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id"`
	DeliveryService   string    `json:"delivery_service"`
	ShardKey          string    `json:"shardkey"`
	SMID              int       `json:"sm_id"`
	DateCreated       time.Time `json:"date_created"`
	OOFShard          string    `json:"oof_shard"`
}

func (o *Order) Validate() error {
	// Проверка OrderUID (20 hex-символов)
	if matched, _ := regexp.MatchString(`^[a-f0-9]{20}$`, o.OrderUID); !matched {
		return errors.New("order_uid должен быть 20-значным hex-идентификатором")
	}

	// Простые проверки без regexp для производительности
	if len(o.TrackNumber) < 10 || len(o.TrackNumber) > 20 {
		return errors.New("track_number должен быть 10-20 символов")
	}

	if o.Locale != "en" && o.Locale != "ru" {
		return errors.New("поддерживаются только локали en/ru")
	}

	// Валидация вложенных структур
	if err := o.Delivery.Validate(); err != nil {
		return err
	}

	if err := o.Payment.Validate(o.OrderUID); err != nil {
		return err
	}

	if len(o.Items) == 0 {
		return errors.New("заказ должен содержать минимум 1 товар")
	}

	for _, item := range o.Items {
		if err := item.Validate(); err != nil {
			return err
		}
	}

	return nil
}
