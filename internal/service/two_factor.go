package service

import (
	"bank-api/internal/errors"
	"bank-api/internal/repository"
	"context"

	"github.com/pquerna/otp/totp"
)

// validateTwoFactorIfEnabled требует код только у пользователей с включённой 2FA
func validateTwoFactorIfEnabled(ctx context.Context, userRepo repository.UserRepo, userID int64, code string) error {
	user, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.TwoFactorEnabled {
		return nil
	}

	if code == "" {
		return errors.NewAppError(401, "Для операции требуется код 2FA", nil)
	}

	if !totp.Validate(code, user.TwoFactorKey) {
		return errors.NewAppError(401, "Неверный код 2FA", nil)
	}

	return nil
}
