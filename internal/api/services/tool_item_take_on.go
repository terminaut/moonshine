package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type ToolItemTakeOnService struct {
	db                *sqlx.DB
	toolItemRepo      *repository.ToolItemRepository
	inventoryRepo     *repository.InventoryRepository
	equipmentItemRepo *repository.EquipmentItemRepository
	userRepo          *repository.UserRepository
}

func NewToolItemTakeOnService(
	db *sqlx.DB,
	toolItemRepo *repository.ToolItemRepository,
	inventoryRepo *repository.InventoryRepository,
	equipmentItemRepo *repository.EquipmentItemRepository,
	userRepo *repository.UserRepository,
) *ToolItemTakeOnService {
	return &ToolItemTakeOnService{
		db:                db,
		toolItemRepo:      toolItemRepo,
		inventoryRepo:     inventoryRepo,
		equipmentItemRepo: equipmentItemRepo,
		userRepo:          userRepo,
	}
}

func (s *ToolItemTakeOnService) TakeOnToolItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) error {
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
	err = tx.Get(&count, `
		SELECT COUNT(*)
		FROM inventory
		WHERE user_id = $1 AND tool_item_id = $2 AND deleted_at IS NULL
	`, userID, itemID)
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrItemNotInInventory
	}

	_, err = tx.Exec(`
		DELETE FROM inventory
		WHERE id = (
			SELECT id FROM inventory
			WHERE user_id = $1 AND tool_item_id = $2 AND deleted_at IS NULL
			LIMIT 1
		)
	`, userID, itemID)
	if err != nil {
		return err
	}

	if user.WeaponEquipmentItemID != nil {
		weapon, err := s.equipmentItemRepo.FindByID(*user.WeaponEquipmentItemID)
		if err != nil {
			return err
		}

		invRepo := repository.NewInventoryRepository(tx)
		err = invRepo.Create(&domain.Inventory{
			UserID:          userID,
			EquipmentItemID: user.WeaponEquipmentItemID,
		})
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			UPDATE users
			SET weapon_equipment_item_id = NULL,
				attack = attack - $2,
				defense = defense - $3,
				hp = hp - $4,
				current_hp = LEAST(current_hp, hp - $4)
			WHERE id = $1 AND deleted_at IS NULL
		`, userID, weapon.Attack, weapon.Defense, weapon.Hp)
		if err != nil {
			return err
		}
	}

	if user.ToolItemID != nil {
		invRepo := repository.NewInventoryRepository(tx)
		err = invRepo.Create(&domain.Inventory{
			UserID:     userID,
			ToolItemID: user.ToolItemID,
		})
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`
		UPDATE users SET tool_item_id = $1 WHERE id = $2 AND deleted_at IS NULL
	`, itemID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
