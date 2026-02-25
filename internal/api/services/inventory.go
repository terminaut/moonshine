package services

import (
	"context"

	"github.com/google/uuid"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type InventoryService struct {
	inventoryRepo *repository.InventoryRepository
}

func NewInventoryService(inventoryRepo *repository.InventoryRepository) *InventoryService {
	return &InventoryService{
		inventoryRepo: inventoryRepo,
	}
}

func (s *InventoryService) GetUserInventory(ctx context.Context, userID uuid.UUID) ([]*domain.EquipmentItem, error) {
	return s.inventoryRepo.FindByUserID(userID)
}

func (s *InventoryService) GetUserInventoryPotions(ctx context.Context, userID uuid.UUID) ([]*domain.Potion, error) {
	return s.inventoryRepo.FindPotionsByUserID(userID)
}
