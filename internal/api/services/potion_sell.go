package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/repository"
)

type PotionSellService struct {
	db         *sqlx.DB
	potionRepo *repository.PotionRepository
	userRepo   *repository.UserRepository
}

func NewPotionSellService(
	db *sqlx.DB,
	potionRepo *repository.PotionRepository,
	userRepo *repository.UserRepository,
) *PotionSellService {
	return &PotionSellService{
		db:         db,
		potionRepo: potionRepo,
		userRepo:   userRepo,
	}
}

func (s *PotionSellService) SellPotion(ctx context.Context, userID uuid.UUID, potionSlug string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	potion, err := s.potionRepo.FindBySlug(potionSlug)
	if err != nil {
		return ErrPotionNotFound
	}

	invRepo := repository.NewInventoryRepository(tx)
	hasPotion, err := invRepo.HasPotion(userID, potion.ID)
	if err != nil {
		return err
	}
	if !hasPotion {
		return ErrItemNotOwned
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	// Potion sell price is 90% of base price (10% lower).
	sellPrice := potion.Price * 9 / 10
	newGold := user.Gold + sellPrice
	if err = s.userRepo.UpdateGoldWithExt(tx, userID, newGold); err != nil {
		return err
	}

	if err = invRepo.DeleteOnePotion(userID, potion.ID); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}
