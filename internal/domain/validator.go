package domain

import (
	"errors"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var (
	validate      *validator.Validate
	orderUIDRegex = regexp.MustCompile(`^[a-f0-9]{20}$`)
	phoneRegex    = regexp.MustCompile(`^\+[0-9]{7,15}$`)
	zipRegex      = regexp.MustCompile(`^[0-9]+$`)
	currencyRegex = regexp.MustCompile(`^[A-Z]{3}$`)
)

func init() {
	validate = validator.New()
	_ = validate.RegisterValidation("order_uid", validateOrderUID)
	_ = validate.RegisterValidation("track_number", validateTrackNumber)
	_ = validate.RegisterValidation("phone", validatePhone)
	_ = validate.RegisterValidation("zip", validateZip)
	_ = validate.RegisterValidation("currency", validateCurrency)
}

// Валидаторы для кастомных полей
func validateOrderUID(fl validator.FieldLevel) bool {
	re := orderUIDRegex
	return re.MatchString(fl.Field().String())
}

func validateTrackNumber(fl validator.FieldLevel) bool {
	return len(fl.Field().String()) >= 10 && len(fl.Field().String()) <= 20
}

func validatePhone(fl validator.FieldLevel) bool {
	re := phoneRegex
	return re.MatchString(fl.Field().String())
}

func validateZip(fl validator.FieldLevel) bool {
	re := zipRegex
	return re.MatchString(fl.Field().String())
}

func validateCurrency(fl validator.FieldLevel) bool {
	re := currencyRegex
	return re.MatchString(fl.Field().String())
}

func (o *Order) Validate() error {
	if o.Payment.Amount != o.Payment.DeliveryCost+o.Payment.GoodsTotal {
		return errors.New("amount должен быть равен delivery_cost + goods_total")
	}
	return validate.Struct(o)
}
