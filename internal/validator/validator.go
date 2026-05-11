package validator

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validate хранит единый валидатор для всех обработчиков
var Validate = validator.New()

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
	digitPattern    = regexp.MustCompile(`[0-9]`)
	letterPattern   = regexp.MustCompile(`[A-Za-zА-Яа-я]`)
)

func init() {
	// в username допускаются только буквы, цифры и подчёркивания
	_ = Validate.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		return usernamePattern.MatchString(fl.Field().String())
	})

	// в password должны быть буквы и цифры, длина от 8 до 50 символов
	_ = Validate.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		password := fl.Field().String()
		if len(password) < 8 || len(password) > 50 {
			return false
		}

		return digitPattern.MatchString(password) && letterPattern.MatchString(password)
	})
}

// HumanizeError переводит технические ошибки валидации в понятный текст для пользователя
func HumanizeError(err error) string {
	if err == nil {
		return ""
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		return err.Error()
	}

	messages := make([]string, 0, len(validationErrs))
	for _, fieldErr := range validationErrs {
		messages = append(messages, translateFieldError(fieldErr))
	}

	return strings.Join(messages, "; ")
}

func translateFieldError(err validator.FieldError) string {
	field := fieldName(err.Field())

	switch err.Tag() {
	case "required":
		return fmt.Sprintf("Поле %s обязательно для заполнения", field)
	case "email":
		return "Укажите корректный email"
	case "min":
		return fmt.Sprintf("Поле %s должно содержать не меньше %s символов", field, err.Param())
	case "max":
		return fmt.Sprintf("Поле %s должно содержать не больше %s символов", field, err.Param())
	case "len":
		return fmt.Sprintf("Поле %s должно содержать ровно %s символов", field, err.Param())
	case "numeric":
		return fmt.Sprintf("Поле %s должно состоять только из цифр", field)
	case "alphanum":
		return fmt.Sprintf("Поле %s может содержать только латинские буквы и цифры", field)
	case "gt":
		return fmt.Sprintf("Поле %s должно быть больше %s", field, err.Param())
	case "eq":
		return fmt.Sprintf("Поле %s должно быть равно %s", field, err.Param())
	case "username":
		return "Username может содержать только латинские буквы, цифры и подчёркивания, длина от 3 до 50 символов"
	case "password":
		return "Пароль должен быть длиной от 8 до 50 символов и содержать хотя бы одну букву и одну цифру"
	default:
		return fmt.Sprintf("Поле %s заполнено некорректно", field)
	}
}

func fieldName(name string) string {
	switch name {
	case "Username":
		return "username"
	case "Email":
		return "email"
	case "Password":
		return "password"
	case "Currency":
		return "currency"
	case "Amount":
		return "amount"
	case "AccountID":
		return "account_id"
	case "FromAccountID":
		return "from_account_id"
	case "ToAccountID":
		return "to_account_id"
	case "TermMonths":
		return "term_months"
	case "CVV":
		return "cvv"
	case "Description":
		return "description"
	case "OTPCode":
		return "otp_code"
	case "Code":
		return "code"
	default:
		return strings.ToLower(name)
	}
}
