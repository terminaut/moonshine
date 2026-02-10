package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/repository"
)

var (
	ErrItemNotOwned = errors.New("item not owned by user")
)

type EquipmentItemSellService struct {
	db                *sqlx.DB
	equipmentItemRepo *repository.EquipmentItemRepository
	inventoryRepo     *repository.InventoryRepository
	userRepo          *repository.UserRepository
}

func NewEquipmentItemSellService(
	db *sqlx.DB,
	equipmentItemRepo *repository.EquipmentItemRepository,
	inventoryRepo *repository.InventoryRepository,
	userRepo *repository.UserRepository,
) *EquipmentItemSellService {
	return &EquipmentItemSellService{
		db:                db,
		equipmentItemRepo: equipmentItemRepo,
		inventoryRepo:     inventoryRepo,
		userRepo:          userRepo,
	}
}

func (s *EquipmentItemSellService) SellEquipmentItem(ctx context.Context, userID uuid.UUID, itemSlug string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	item, err := s.equipmentItemRepo.FindBySlug(itemSlug)
	if err != nil {
		return ErrEquipmentItemNotFound
	}

	var count int
	checkOwnershipQuery := `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND equipment_item_id = $2 AND deleted_at IS NULL`
	err = tx.Get(&count, checkOwnershipQuery, userID, item.ID)
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
	updateGoldQuery := `UPDATE users SET gold = $1 WHERE id = $2`
	_, err = tx.Exec(updateGoldQuery, newGold, userID)
	if err != nil {
		return err
	}

	deleteItemQuery := `DELETE FROM inventory WHERE user_id = $1 AND equipment_item_id = $2 AND deleted_at IS NULL`
	_, err = tx.Exec(deleteItemQuery, userID, item.ID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
