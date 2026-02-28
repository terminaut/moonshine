package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type PotionBuyService struct {
	db            *sqlx.DB
	potionRepo    *repository.PotionRepository
	inventoryRepo *repository.InventoryRepository
	userRepo      *repository.UserRepository
}

func NewPotionBuyService(
	db *sqlx.DB,
	potionRepo *repository.PotionRepository,
	inventoryRepo *repository.InventoryRepository,
	userRepo *repository.UserRepository,
) *PotionBuyService {
	return &PotionBuyService{
		db:            db,
		potionRepo:    potionRepo,
		inventoryRepo: inventoryRepo,
		userRepo:      userRepo,
	}
}

func (s *PotionBuyService) BuyPotion(ctx context.Context, userID uuid.UUID, potionSlug string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	potion, err := s.potionRepo.FindBySlug(potionSlug)
	if err != nil {
		return repository.ErrPotionNotFound
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	if user.Gold < potion.Price {
		return ErrInsufficientGold
	}

	inventory := &domain.Inventory{
		UserID:   userID,
		PotionID: &potion.ID,
	}

	inventoryRepo := repository.NewInventoryRepository(tx)
	if err := inventoryRepo.Create(inventory); err != nil {
		return err
	}

	newGold := user.Gold - potion.Price
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
