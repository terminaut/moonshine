package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/repository"
)

type ToolItemSellService struct {
	db           *sqlx.DB
	toolItemRepo *repository.ToolItemRepository
	userRepo     *repository.UserRepository
}

func NewToolItemSellService(
	db *sqlx.DB,
	toolItemRepo *repository.ToolItemRepository,
	userRepo *repository.UserRepository,
) *ToolItemSellService {
	return &ToolItemSellService{
		db:           db,
		toolItemRepo: toolItemRepo,
		userRepo:     userRepo,
	}
}

func (s *ToolItemSellService) SellToolItem(ctx context.Context, userID uuid.UUID, itemSlug string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	item, err := s.toolItemRepo.FindBySlug(itemSlug)
	if err != nil {
		return repository.ErrToolItemNotFound
	}

	var count int
	err = tx.Get(&count, `
		SELECT COUNT(*)
		FROM inventory
		WHERE user_id = $1 AND tool_item_id = $2 AND deleted_at IS NULL
	`, userID, item.ID)
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrItemNotOwned
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	newGold := user.Gold + item.Price
	_, err = tx.Exec(`UPDATE users SET gold = $1 WHERE id = $2 AND deleted_at IS NULL`, newGold, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		DELETE FROM inventory
		WHERE id = (
			SELECT id
			FROM inventory
			WHERE user_id = $1 AND tool_item_id = $2 AND deleted_at IS NULL
			LIMIT 1
		)
	`, userID, item.ID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
