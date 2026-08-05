package api

import (
	"github.com/MumAroi/go-simplebank/util"
	"github.com/go-playground/validator/v10"
)

var validCurrencies validator.Func = func(fieldLevel validator.FieldLevel) bool {
	if currency, ok := fieldLevel.Field().Interface().(string); ok {
		return util.IsSupportedCurrency(currency)
	}

	return false
}
