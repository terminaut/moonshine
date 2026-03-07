package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type ToolItemBuyService struct {
	db            *sqlx.DB
	toolItemRepo  *repository.ToolItemRepository
	inventoryRepo *repository.InventoryRepository
	userRepo      *repository.UserRepository
}

func NewToolItemBuyService(
	db *sqlx.DB,
	toolItemRepo *repository.ToolItemRepository,
	inventoryRepo *repository.InventoryRepository,
	userRepo *repository.UserRepository,
) *ToolItemBuyService {
	return &ToolItemBuyService{
		db:            db,
		toolItemRepo:  toolItemRepo,
		inventoryRepo: inventoryRepo,
		userRepo:      userRepo,
	}
}

func (s *ToolItemBuyService) BuyToolItem(ctx context.Context, userID uuid.UUID, itemSlug string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	item, err := s.toolItemRepo.FindBySlug(itemSlug)
	if err != nil {
		return repository.ErrToolItemNotFound
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	if user.Gold < item.Price {
		return ErrInsufficientGold
	}

	inventory := &domain.Inventory{
		UserID:     userID,
		ToolItemID: &item.ID,
	}

	inventoryRepo := repository.NewInventoryRepository(tx)
	if err := inventoryRepo.Create(inventory); err != nil {
		return err
	}

	newGold := user.Gold - item.Price
	updateQuery := `UPDATE users SET gold = $1 WHERE id = $2 AND deleted_at IS NULL`
	_, err = tx.Exec(updateQuery, newGold, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
