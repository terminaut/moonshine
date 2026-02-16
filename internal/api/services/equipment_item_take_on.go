package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

var (
	ErrItemNotInInventory   = errors.New("item not in inventory")
	ErrInsufficientLevel    = errors.New("insufficient level")
	ErrInvalidEquipmentType = errors.New("invalid equipment type")
)

type EquipmentItemTakeOnService struct {
	db                *sqlx.DB
	equipmentItemRepo *repository.EquipmentItemRepository
	inventoryRepo     *repository.InventoryRepository
	userRepo          *repository.UserRepository
}

func NewEquipmentItemTakeOnService(
	db *sqlx.DB,
	equipmentItemRepo *repository.EquipmentItemRepository,
	inventoryRepo *repository.InventoryRepository,
	userRepo *repository.UserRepository,
) *EquipmentItemTakeOnService {
	return &EquipmentItemTakeOnService{
		db:                db,
		equipmentItemRepo: equipmentItemRepo,
		inventoryRepo:     inventoryRepo,
		userRepo:          userRepo,
	}
}

func getEquipmentFieldName(equipmentType string) (string, error) {
	fieldMap := map[string]string{
		"chest":  "chest_equipment_item_id",
		"belt":   "belt_equipment_item_id",
		"head":   "head_equipment_item_id",
		"neck":   "neck_equipment_item_id",
		"weapon": "weapon_equipment_item_id",
		"shield": "shield_equipment_item_id",
		"legs":   "legs_equipment_item_id",
		"feet":   "feet_equipment_item_id",
		"arms":   "arms_equipment_item_id",
		"hands":  "hands_equipment_item_id",
	}

	if fieldName, ok := fieldMap[equipmentType]; ok {
		return fieldName, nil
	}

	if equipmentType == "ring" {
		return "ring", nil
	}

	return "", ErrInvalidEquipmentType
}

func (s *EquipmentItemTakeOnService) TakeOnEquipmentItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	checkQuery := `
		SELECT COUNT(*) 
		FROM inventory 
		WHERE user_id = $1 AND equipment_item_id = $2 AND deleted_at IS NULL
	`
	var count int
	err = tx.Get(&count, checkQuery, userID, itemID)
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrItemNotInInventory
	}

	item, err := s.equipmentItemRepo.FindByID(itemID)
	if err != nil {
		return ErrEquipmentItemNotFound
	}

	categoryQuery := `SELECT type FROM equipment_categories WHERE id = $1 AND deleted_at IS NULL`
	var equipmentType string
	err = tx.Get(&equipmentType, categoryQuery, item.EquipmentCategoryID)
	if err != nil {
		return err
	}

	if user.Level < item.RequiredLevel {
		return ErrInsufficientLevel
	}

	fieldName, err := getEquipmentFieldName(equipmentType)
	if err != nil {
		return err
	}

	var oldItemID *uuid.UUID

	if equipmentType == "ring" {
		switch {
		case user.Ring1EquipmentItemID == nil:
			fieldName = "ring1_equipment_item_id"
		case user.Ring2EquipmentItemID == nil:
			fieldName = "ring2_equipment_item_id"
		case user.Ring3EquipmentItemID == nil:
			fieldName = "ring3_equipment_item_id"
		case user.Ring4EquipmentItemID == nil:
			fieldName = "ring4_equipment_item_id"
		default:
			fieldName = "ring1_equipment_item_id"
			oldItemID = user.Ring1EquipmentItemID
		}
	} else {
		getOldItemQuery := fmt.Sprintf(`
			SELECT %s 
			FROM users 
			WHERE id = $1 AND deleted_at IS NULL
		`, fieldName)
		err = tx.Get(&oldItemID, getOldItemQuery, userID)
		if err != nil {
			oldItemID = nil
		}
	}

	deleteFromInventoryQuery := `
		DELETE FROM inventory 
		WHERE id = (
			SELECT id FROM inventory 
			WHERE user_id = $1 AND equipment_item_id = $2 AND deleted_at IS NULL 
			LIMIT 1
		)
	`
	_, err = tx.Exec(deleteFromInventoryQuery, userID, itemID)
	if err != nil {
		return err
	}

	if oldItemID != nil {
		inventory := &domain.Inventory{
			UserID:          userID,
			EquipmentItemID: *oldItemID,
		}
		inventoryRepo := repository.NewInventoryRepository(tx)
		err = inventoryRepo.Create(inventory)
		if err != nil {
			return err
		}
	}

	var oldItem *domain.EquipmentItem
	if oldItemID != nil {
		oldItem, err = s.equipmentItemRepo.FindByID(*oldItemID)
		if err != nil {
			return err
		}
	}

	var updateStatsQuery string
	if oldItem != nil {
		updateStatsQuery = fmt.Sprintf(`
			UPDATE users 
			SET %s = $1,
				attack = attack - $2 + $5,
				defense = defense - $3 + $6,
				hp = hp - $4 + $7,
				current_hp = LEAST(current_hp, hp - $4 + $7)
			WHERE id = $8 AND deleted_at IS NULL
		`, fieldName)
		_, err = tx.Exec(updateStatsQuery, itemID, oldItem.Attack, oldItem.Defense, oldItem.Hp, item.Attack, item.Defense, item.Hp, userID)
	} else {
		updateStatsQuery = fmt.Sprintf(`
			UPDATE users 
			SET %s = $1,
				attack = attack + $2,
				defense = defense + $3,
				hp = hp + $4,
				current_hp = LEAST(current_hp, hp + $4)
			WHERE id = $5 AND deleted_at IS NULL
		`, fieldName)
		_, err = tx.Exec(updateStatsQuery, itemID, item.Attack, item.Defense, item.Hp, userID)
	}
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
