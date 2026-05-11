package service

import (
	"bank-api/internal/dto"
	"bank-api/internal/errors"
	"bank-api/internal/repository"
	"context"
)

// AdminService отвечает за логику работы функций для администратора
type AdminService interface {
	ListUsers(ctx context.Context) ([]*dto.AdminUserResponse, error)
	BlockUser(ctx context.Context, userID int64, block bool) error
}

// adminService работает с пользователями от имени администратора
type adminService struct {
	userRepo repository.UserRepo
}

// NewAdminService создаёт сервис, отвечающий за логику работы функций для администратора
func NewAdminService(repos *repository.Repositories) AdminService {
	return &adminService{userRepo: repos.User}
}

// ListUsers создаёт список с информацией по всем пользователям
func (s *adminService) ListUsers(ctx context.Context) ([]*dto.AdminUserResponse, error) {
	users, err := s.userRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	var res []*dto.AdminUserResponse
	for _, u := range users {
		res = append(res, &dto.AdminUserResponse{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			IsBlocked: u.IsBlocked,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
		})
	}
	return res, nil
}

// BlockUser включает или снимает блокировку выбранного пользователя
func (s *adminService) BlockUser(ctx context.Context, userID int64, block bool) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.WrapNotFound(err, "Пользователь не найден")
	}
	if user.Role == "admin" && block {
		return errors.NewAppError(400, "Нельзя заблокировать администратора", nil)
	}
	user.IsBlocked = block
	return s.userRepo.Update(ctx, user)
}
