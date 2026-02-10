package services

import (
	"context"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type AvatarService struct {
	avatarRepo *repository.AvatarRepository
}

func NewAvatarService(avatarRepo *repository.AvatarRepository) *AvatarService {
	return &AvatarService{
		avatarRepo: avatarRepo,
	}
}

func (s *AvatarService) GetAllAvatars(ctx context.Context) ([]*domain.Avatar, error) {
	avatars, err := s.avatarRepo.FindAll()
	if err != nil {
		return nil, err
	}
	return avatars, nil
}
