package dto

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/go-playground/validator/v10"
)

// MaxTextBytes is the maximum payload of a MariaDB TEXT column in bytes.
const MaxTextBytes = 65535

// RegisterValidations registers the custom validations used by API DTOs.
func RegisterValidations(v *validator.Validate) error {
	if err := v.RegisterValidation("maxbytes", maxBytes); err != nil {
		return fmt.Errorf("register maxbytes validation: %w", err)
	}

	return nil
}

func maxBytes(fl validator.FieldLevel) bool {
	limit, err := strconv.Atoi(fl.Param())
	if err != nil || fl.Field().Kind() != reflect.String {
		return false
	}

	return len(fl.Field().String()) <= limit
}
