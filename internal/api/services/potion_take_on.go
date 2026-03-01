package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type PotionTakeOnService struct {
	db            *sqlx.DB
	potionRepo    *repository.PotionRepository
	inventoryRepo *repository.InventoryRepository
	userRepo      *repository.UserRepository
}

func NewPotionTakeOnService(
	db *sqlx.DB,
	potionRepo *repository.PotionRepository,
	inventoryRepo *repository.InventoryRepository,
	userRepo *repository.UserRepository,
) *PotionTakeOnService {
	return &PotionTakeOnService{
		db:            db,
		potionRepo:    potionRepo,
		inventoryRepo: inventoryRepo,
		userRepo:      userRepo,
	}
}

func (s *PotionTakeOnService) TakeOnPotion(ctx context.Context, userID uuid.UUID, potionID uuid.UUID) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	var count int
	checkQuery := `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND potion_id = $2 AND deleted_at IS NULL`
	err = tx.Get(&count, checkQuery, userID, potionID)
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrItemNotInInventory
	}

	_, err = s.potionRepo.FindByID(potionID)
	if err != nil {
		return repository.ErrPotionNotFound
	}

	var fieldName string
	var oldPotionID *uuid.UUID

	switch {
	case user.Potion1ID == nil:
		fieldName = "potion1_id"
	case user.Potion2ID == nil:
		fieldName = "potion2_id"
	case user.Potion3ID == nil:
		fieldName = "potion3_id"
	default:
		fieldName = "potion1_id"
		oldPotionID = user.Potion1ID
	}

	deleteFromInventoryQuery := `
		DELETE FROM inventory
		WHERE id = (
			SELECT id FROM inventory
			WHERE user_id = $1 AND potion_id = $2 AND deleted_at IS NULL
			LIMIT 1
		)
	`
	_, err = tx.Exec(deleteFromInventoryQuery, userID, potionID)
	if err != nil {
		return err
	}

	if oldPotionID != nil {
		inventory := &domain.Inventory{
			UserID:   userID,
			PotionID: oldPotionID,
		}
		inventoryRepo := repository.NewInventoryRepository(tx)
		err = inventoryRepo.Create(inventory)
		if err != nil {
			return err
		}
	}

	updateQuery := fmt.Sprintf(`
		UPDATE users
		SET %s = $1
		WHERE id = $2 AND deleted_at IS NULL
	`, fieldName)
	_, err = tx.Exec(updateQuery, potionID, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
