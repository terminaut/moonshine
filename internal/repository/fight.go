package repository

import (
	"github.com/google/uuid"

	"moonshine/internal/domain"
)

type FightRepository struct {
	db ExtHandle
}

func NewFightRepository(db ExtHandle) *FightRepository {
	return &FightRepository{db: db}
}

func (r *FightRepository) Create(fight *domain.Fight) (uuid.UUID, error) {
	status := fight.Status
	if status == "" {
		status = domain.FightStatusInProgress
	}

	query := `
		INSERT INTO fights (user_id, bot_id, status)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	err := r.db.QueryRow(query,
		fight.UserID, fight.BotID, status,
	).Scan(&fight.ID)
	if err != nil {
		return fight.ID, err
	}
	fight.Status = status
	return fight.ID, err
}

func (r *FightRepository) FindActiveByUserID(userID uuid.UUID) (*domain.Fight, error) {
	query := `
		SELECT id, created_at, deleted_at, user_id, bot_id, status, dropped_gold, exp, dropped_item_id
		FROM fights
		WHERE user_id = $1 AND status = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`

	fight := &domain.Fight{}
	err := r.db.Get(fight, query, userID, domain.FightStatusInProgress)
	if err != nil {
		return nil, err
	}

	return fight, nil
}

func (r *FightRepository) Finish(id uuid.UUID, droppedGold, exp uint) (*domain.Fight, error) {
	query := `
		UPDATE fights
		SET status = $1,
		    dropped_gold = $2,
		    exp = $3
		WHERE id = $4
		RETURNING id, created_at, deleted_at, user_id, bot_id, status, dropped_gold, exp, dropped_item_id
	`

	fight := &domain.Fight{}
	err := r.db.Get(fight, query, string(domain.FightStatusFinished), droppedGold, exp, id)
	if err != nil {
		return nil, err
	}

	return fight, nil
}
