package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type PotionTakeOffService struct {
	db            *sqlx.DB
	potionRepo    *repository.PotionRepository
	inventoryRepo *repository.InventoryRepository
	userRepo      *repository.UserRepository
}

func NewPotionTakeOffService(
	db *sqlx.DB,
	potionRepo *repository.PotionRepository,
	inventoryRepo *repository.InventoryRepository,
	userRepo *repository.UserRepository,
) *PotionTakeOffService {
	return &PotionTakeOffService{
		db:            db,
		potionRepo:    potionRepo,
		inventoryRepo: inventoryRepo,
		userRepo:      userRepo,
	}
}

func getPotionFieldNameFromSlot(slotName string) (string, error) {
	fieldMap := map[string]string{
		"potion1": "potion1_id",
		"potion2": "potion2_id",
		"potion3": "potion3_id",
	}

	if fieldName, ok := fieldMap[slotName]; ok {
		return fieldName, nil
	}

	return "", ErrInvalidEquipmentType
}

func (s *PotionTakeOffService) TakeOffPotion(ctx context.Context, userID uuid.UUID, slotName string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	fieldName, err := getPotionFieldNameFromSlot(slotName)
	if err != nil {
		return err
	}

	getItemQuery := fmt.Sprintf(`
		SELECT %s
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, fieldName)

	var equippedPotionIDStr sql.NullString
	err = tx.Get(&equippedPotionIDStr, getItemQuery, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.ErrUserNotFound
		}
		return err
	}

	if !equippedPotionIDStr.Valid || equippedPotionIDStr.String == "" {
		return ErrNoItemEquipped
	}

	equippedPotionID, err := uuid.Parse(equippedPotionIDStr.String)
	if err != nil {
		return err
	}

	invRepo := repository.NewInventoryRepository(tx)
	err = invRepo.Create(&domain.Inventory{UserID: userID, PotionID: &equippedPotionID})
	if err != nil {
		return err
	}

	clearSlotQuery := fmt.Sprintf(`
		UPDATE users
		SET %s = NULL
		WHERE id = $1 AND deleted_at IS NULL
	`, fieldName)
	_, err = tx.Exec(clearSlotQuery, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
