package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

var (
	ErrInsufficientGold = errors.New("insufficient gold")
)

type EquipmentItemBuyService struct {
	db                *sqlx.DB
	equipmentItemRepo *repository.EquipmentItemRepository
	inventoryRepo     *repository.InventoryRepository
	userRepo          *repository.UserRepository
}

func NewEquipmentItemBuyService(
	db *sqlx.DB,
	equipmentItemRepo *repository.EquipmentItemRepository,
	inventoryRepo *repository.InventoryRepository,
	userRepo *repository.UserRepository,
) *EquipmentItemBuyService {
	return &EquipmentItemBuyService{
		db:                db,
		equipmentItemRepo: equipmentItemRepo,
		inventoryRepo:     inventoryRepo,
		userRepo:          userRepo,
	}
}

func (s *EquipmentItemBuyService) BuyEquipmentItem(ctx context.Context, userID uuid.UUID, itemSlug string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	item, err := s.equipmentItemRepo.FindBySlug(itemSlug)
	if err != nil {
		return repository.ErrEquipmentItemNotFound
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	if user.Gold < item.Price {
		return ErrInsufficientGold
	}

	inventory := &domain.Inventory{
		UserID:          userID,
		EquipmentItemID: &item.ID,
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
