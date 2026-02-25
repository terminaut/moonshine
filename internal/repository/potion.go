package repository

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"moonshine/internal/domain"
)

var (
	ErrPotionNotFound = errors.New("potion not found")
)

type PotionRepository struct {
	db *sqlx.DB
}

func NewPotionRepository(db *sqlx.DB) *PotionRepository {
	return &PotionRepository{db: db}
}

func (r *PotionRepository) FindAll() ([]*domain.Potion, error) {
	query := `
		SELECT id, created_at, deleted_at, name, slug, stat, value, price, COALESCE(image, '') as image
		FROM potions
		WHERE deleted_at IS NULL
		ORDER BY price ASC
	`

	var potions []*domain.Potion
	err := r.db.Select(&potions, query)
	if err != nil {
		return nil, err
	}

	return potions, nil
}

func (r *PotionRepository) FindBySlug(slug string) (*domain.Potion, error) {
	query := `
		SELECT id, created_at, deleted_at, name, slug, stat, value, price, COALESCE(image, '') as image
		FROM potions
		WHERE slug = $1 AND deleted_at IS NULL
	`

	potion := &domain.Potion{}
	err := r.db.Get(potion, query, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPotionNotFound
		}
		return nil, err
	}

	return potion, nil
}

func (r *PotionRepository) FindByID(id uuid.UUID) (*domain.Potion, error) {
	query := `
		SELECT id, created_at, deleted_at, name, slug, stat, value, price, COALESCE(image, '') as image
		FROM potions
		WHERE id = $1 AND deleted_at IS NULL
	`

	potion := &domain.Potion{}
	err := r.db.Get(potion, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPotionNotFound
		}
		return nil, err
	}

	return potion, nil
}

func (r *PotionRepository) FindByIDs(ids []uuid.UUID) ([]*domain.Potion, error) {
	if len(ids) == 0 {
		return []*domain.Potion{}, nil
	}

	query := `
		SELECT id, created_at, deleted_at, name, slug, stat, value, price, COALESCE(image, '') as image
		FROM potions
		WHERE id = ANY($1) AND deleted_at IS NULL
	`

	var potions []*domain.Potion
	if err := r.db.Select(&potions, query, pq.Array(ids)); err != nil {
		return nil, err
	}

	return potions, nil
}
