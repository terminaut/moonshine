package repository

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
)

var (
	ErrResourceNotFound = errors.New("resource not found")
)

type ResourceRepository struct {
	db *sqlx.DB
}

func NewResourceRepository(db *sqlx.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

func (r *ResourceRepository) FindResourcesByLocationID(locationID uuid.UUID) ([]*domain.Resource, error) {
	query := `
		SELECT r.id, r.created_at, r.deleted_at, r.name, r.slug, r.tool_category_id, r.tool_item_id, COALESCE(r.image, '') AS image
		FROM resources r
		INNER JOIN location_resources lr ON lr.resource_id = r.id
		WHERE lr.location_id = $1 AND lr.deleted_at IS NULL AND r.deleted_at IS NULL
		ORDER BY r.name ASC
	`

	var resources []*domain.Resource
	if err := r.db.Select(&resources, query, locationID); err != nil {
		return nil, err
	}

	return resources, nil
}

func (r *ResourceRepository) FindBySlug(slug string) (*domain.Resource, error) {
	query := `
		SELECT r.id, r.deleted_at, name, slug, tool_category_id, tool_item_id, location_id as location_id, COALESCE(image, '') AS image
		FROM resources r
		JOIN location_resources lr ON lr.resource_id = r.id
		WHERE slug = $1 AND r.deleted_at IS NULL;
	`

	resource := &domain.Resource{}
	if err := r.db.Get(resource, query, slug); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrResourceNotFound
		}
		return nil, err
	}

	return resource, nil
}
