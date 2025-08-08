package validation

import (
	"github.com/go-playground/validator/v10"
	"regexp"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	_ = validate.RegisterValidation("order_uid", validateOrderUID)
	_ = validate.RegisterValidation("track_number", validateTrackNumber)
	_ = validate.RegisterValidation("phone", validatePhone)
	_ = validate.RegisterValidation("zip", validateZip)
	_ = validate.RegisterValidation("currency", validateCurrency)
}

func validateOrderUID(fl validator.FieldLevel) bool {
	re := regexp.MustCompile(`^[a-f0-9]{20}$`)
	return re.MatchString(fl.Field().String())
}

func validateTrackNumber(fl validator.FieldLevel) bool {
	return len(fl.Field().String()) >= 10 && len(fl.Field().String()) <= 20
}

func validatePhone(fl validator.FieldLevel) bool {
	re := regexp.MustCompile(`^\+[0-9]{7,15}$`)
	return re.MatchString(fl.Field().String())
}

func validateZip(fl validator.FieldLevel) bool {
	re := regexp.MustCompile(`^[0-9]+$`)
	return re.MatchString(fl.Field().String())
}

func validateCurrency(fl validator.FieldLevel) bool {
	re := regexp.MustCompile(`^[A-Z]{3}$`)
	return re.MatchString(fl.Field().String())
}
