package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type ToolItemTakeOffService struct {
	db            *sqlx.DB
	inventoryRepo *repository.InventoryRepository
	userRepo      *repository.UserRepository
}

func NewToolItemTakeOffService(
	db *sqlx.DB,
	inventoryRepo *repository.InventoryRepository,
	userRepo *repository.UserRepository,
) *ToolItemTakeOffService {
	return &ToolItemTakeOffService{
		db:            db,
		inventoryRepo: inventoryRepo,
		userRepo:      userRepo,
	}
}

func (s *ToolItemTakeOffService) TakeOffToolItem(ctx context.Context, userID uuid.UUID) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	if user.ToolItemID == nil {
		return ErrNoItemEquipped
	}

	invRepo := repository.NewInventoryRepository(tx)
	err = invRepo.Create(&domain.Inventory{
		UserID:     userID,
		ToolItemID: user.ToolItemID,
	})
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE users SET tool_item_id = NULL WHERE id = $1 AND deleted_at IS NULL
	`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
