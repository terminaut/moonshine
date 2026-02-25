package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type PotionUseService struct {
	db         *sqlx.DB
	potionRepo *repository.PotionRepository
	userRepo   *repository.UserRepository
}

func NewPotionUseService(
	db *sqlx.DB,
	potionRepo *repository.PotionRepository,
	userRepo *repository.UserRepository,
) *PotionUseService {
	return &PotionUseService{
		db:         db,
		potionRepo: potionRepo,
		userRepo:   userRepo,
	}
}

func (s *PotionUseService) UsePotion(ctx context.Context, userID uuid.UUID, slot string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	potionId, err := resolvePotionId(slot, user)
	if err != nil {
		return err
	}

	potion, err := s.potionRepo.FindByID(potionId)
	if err != nil {
		return ErrPotionNotFound
	}

	fightRepo := repository.NewFightRepository(s.db)
	fight, err := fightRepo.FindActiveByUserID(userID)
	if err != nil {
		return ErrNoActiveFight
	}

	roundRepo := repository.NewRoundRepository(s.db)
	rounds, err := roundRepo.FindByFightID(fight.ID)
	if err != nil || len(rounds) == 0 {
		return ErrInternalError
	}

	currentRound := rounds[0]
	newHp := int(calculateCurrentHp(potion.Value, user.Hp, currentRound.PlayerHp))

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	roundRepoTx := repository.NewRoundRepository(tx)
	if err = roundRepoTx.UpdatePlayerHp(currentRound.ID, newHp); err != nil {
		return err
	}

	if err = s.userRepo.ClearPotionSlot(tx, userID, slotToColumn(slot), newHp); err != nil {
		return err
	}

	return tx.Commit()
}

func resolvePotionId(slot string, user *domain.User) (uuid.UUID, error) {
	var id *uuid.UUID

	switch slot {
	case "potion1":
		id = user.Potion1ID
	case "potion2":
		id = user.Potion2ID
	case "potion3":
		id = user.Potion3ID
	default:
		return uuid.Nil, ErrInvalidEquipmentType
	}

	if id == nil {
		return uuid.Nil, ErrNoItemEquipped
	}

	return *id, nil
}

func calculateCurrentHp(value, hp uint, currentHp int) uint {
	newHp := uint(currentHp) + value
	if newHp > hp {
		return hp
	}
	return newHp
}

func slotToColumn(slot string) string {
	switch slot {
	case "potion1":
		return "potion1_id"
	case "potion2":
		return "potion2_id"
	case "potion3":
		return "potion3_id"
	default:
		return "potion1_id"
	}
}
