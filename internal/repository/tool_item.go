package repository

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
)

var (
	ErrToolItemNotFound = errors.New("tool item not found")
)

type ToolItemRepository struct {
	db *sqlx.DB
}

func NewToolItemRepository(db *sqlx.DB) *ToolItemRepository {
	return &ToolItemRepository{db: db}
}

func (r *ToolItemRepository) FindByID(id uuid.UUID) (*domain.ToolItem, error) {
	query := `
		SELECT id, created_at, deleted_at, name, slug, price, tool_category_id, COALESCE(image, '') as image
		FROM tool_items
		WHERE id = $1 AND deleted_at IS NULL
	`

	item := &domain.ToolItem{}
	err := r.db.Get(item, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrToolItemNotFound
		}
		return nil, err
	}

	return item, nil
}

func (r *ToolItemRepository) FindAll() ([]*domain.ToolItem, error) {
	query := `
		SELECT id, created_at, deleted_at, name, slug, price, tool_category_id, COALESCE(image, '') as image
		FROM tool_items
		WHERE deleted_at IS NULL
		ORDER BY price ASC, name ASC
	`

	var items []*domain.ToolItem
	if err := r.db.Select(&items, query); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *ToolItemRepository) FindBySlug(slug string) (*domain.ToolItem, error) {
	query := `
		SELECT id, created_at, deleted_at, name, slug, price, tool_category_id, COALESCE(image, '') as image
		FROM tool_items
		WHERE slug = $1 AND deleted_at IS NULL
	`

	item := &domain.ToolItem{}
	err := r.db.Get(item, query, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrToolItemNotFound
		}
		return nil, err
	}

	return item, nil
}
