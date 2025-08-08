package domain

import (
	"errors"
	"regexp"
)

type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

func (d *Delivery) Validate() error {
	if len(d.Name) < 2 || len(d.Name) > 100 {
		return errors.New("имя доставки должно быть 2-100 символов")
	}

	if matched, _ := regexp.MatchString(`^\+[0-9]{7,15}$`, d.Phone); !matched {
		return errors.New("неверный формат телефона")
	}

	if len(d.Zip) == 0 {
		return errors.New("индекс обязателен")
	}

	return nil
}
