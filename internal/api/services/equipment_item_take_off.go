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

var (
	ErrNoItemEquipped = errors.New("no item equipped in this slot")
)

type EquipmentItemTakeOffService struct {
	db                *sqlx.DB
	equipmentItemRepo *repository.EquipmentItemRepository
	inventoryRepo     *repository.InventoryRepository
	userRepo          *repository.UserRepository
}

func NewEquipmentItemTakeOffService(
	db *sqlx.DB,
	equipmentItemRepo *repository.EquipmentItemRepository,
	inventoryRepo *repository.InventoryRepository,
	userRepo *repository.UserRepository,
) *EquipmentItemTakeOffService {
	return &EquipmentItemTakeOffService{
		db:                db,
		equipmentItemRepo: equipmentItemRepo,
		inventoryRepo:     inventoryRepo,
		userRepo:          userRepo,
	}
}

func getFieldNameFromSlot(slotName string) (string, error) {
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
		"ring1":  "ring1_equipment_item_id",
		"ring2":  "ring2_equipment_item_id",
		"ring3":  "ring3_equipment_item_id",
		"ring4":  "ring4_equipment_item_id",
	}

	if fieldName, ok := fieldMap[slotName]; ok {
		return fieldName, nil
	}

	return "", ErrInvalidEquipmentType
}

func (s *EquipmentItemTakeOffService) TakeOffEquipmentItem(ctx context.Context, userID uuid.UUID, slotName string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	fieldName, err := getFieldNameFromSlot(slotName)
	if err != nil {
		return err
	}

	getItemQuery := fmt.Sprintf(`
		SELECT %s 
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL
	`, fieldName)

	var equippedItemIDStr sql.NullString
	err = tx.Get(&equippedItemIDStr, getItemQuery, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.ErrUserNotFound
		}
		return err
	}

	if !equippedItemIDStr.Valid || equippedItemIDStr.String == "" {
		return ErrNoItemEquipped
	}

	equippedItemID, err := uuid.Parse(equippedItemIDStr.String)
	if err != nil {
		return err
	}

	item, err := s.equipmentItemRepo.FindByID(equippedItemID)
	if err != nil {
		return err
	}

	invRepo := repository.NewInventoryRepository(tx)
	err = invRepo.Create(&domain.Inventory{UserID: userID, EquipmentItemID: equippedItemID})
	if err != nil {
		return err
	}

	clearSlotQuery := fmt.Sprintf(`
		UPDATE users 
		SET %s = NULL,
			attack = attack - $2,
			defense = defense - $3,
			hp = hp - $4,
			current_hp = LEAST(current_hp, hp - $4)
		WHERE id = $1 AND deleted_at IS NULL
	`, fieldName)
	_, err = tx.Exec(clearSlotQuery, userID, item.Attack, item.Defense, item.Hp)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
