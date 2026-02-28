package services

import (
	"context"
	"log"

	"moonshine/internal/domain"
	r "moonshine/internal/redis"
	"moonshine/internal/repository"

	"github.com/google/uuid"
)

type UserService struct {
	userRepo     *repository.UserRepository
	avatarRepo   *repository.AvatarRepository
	locationRepo *repository.LocationRepository
	userCache    r.Cache[domain.User]
}

func NewUserService(
	userRepo *repository.UserRepository,
	avatarRepo *repository.AvatarRepository,
	locationRepo *repository.LocationRepository,
	userCache r.Cache[domain.User],
) *UserService {
	return &UserService{
		userRepo:     userRepo,
		avatarRepo:   avatarRepo,
		locationRepo: locationRepo,
		userCache:    userCache,
	}
}

func (s *UserService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, repository.ErrUserNotFound
	}

	return user, nil
}

func (s *UserService) GetCurrentUserWithRelations(ctx context.Context, userID uuid.UUID) (*domain.User, *domain.Location, bool, error) {
	var user *domain.User
	var err error

	if s.userCache != nil {
		user, err = s.userCache.Get(ctx, userID.String())
		if err != nil {
			log.Printf("[UserService] redis get error: %v", err)
		}
	}

	if user == nil {
		user, err = s.userRepo.FindByID(userID)
		if err != nil {
			return nil, nil, false, repository.ErrUserNotFound
		}

		if s.userCache != nil {
			_ = s.userCache.Set(ctx, userID.String(), user)
		}
	}

	var location *domain.Location
	if s.locationRepo != nil && user.LocationID != uuid.Nil {
		location, _ = s.locationRepo.FindByID(user.LocationID)
	}

	inFight, _ := s.userRepo.InFight(userID)

	return user, location, inFight, nil
}

func (s *UserService) UpdateUser(ctx context.Context, userID uuid.UUID, avatarID *uuid.UUID) (*domain.User, error) {
	if avatarID != nil {
		_, err := s.avatarRepo.FindByID(*avatarID)
		if err != nil {
			return nil, repository.ErrAvatarNotFound
		}
	}

	err := s.userRepo.UpdateAvatarID(userID, avatarID)
	if err != nil {
		return nil, err
	}

	if s.userCache != nil {
		_ = s.userCache.Delete(ctx, userID.String())
	}

	user, err := s.GetCurrentUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if s.userCache != nil {
		_ = s.userCache.Set(ctx, userID.String(), user)
	}

	return user, nil
}
