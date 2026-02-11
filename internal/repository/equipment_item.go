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
	ErrEquipmentItemNotFound = errors.New("equipment item not found")
)

type EquipmentItemRepository struct {
	db *sqlx.DB
}

func NewEquipmentItemRepository(db *sqlx.DB) *EquipmentItemRepository {
	return &EquipmentItemRepository{db: db}
}

func (r *EquipmentItemRepository) FindByCategorySlug(slug string) ([]*domain.EquipmentItem, error) {
	return r.FindByCategorySlugAndArtifact(slug, false)
}

func (r *EquipmentItemRepository) FindByCategorySlugAndArtifact(slug string, artifact bool) ([]*domain.EquipmentItem, error) {
	query := `
		SELECT ei.id, ei.created_at, ei.deleted_at, ei.name, ei.slug, ei.attack, ei.defense, ei.hp,
			ei.required_level, ei.price, ei.artifact, ei.equipment_category_id, ei.image
		FROM equipment_items ei
		INNER JOIN equipment_categories ec ON ei.equipment_category_id = ec.id
		WHERE ec.type = $1::equipment_category_type 
			AND ei.artifact = $2
			AND ei.deleted_at IS NULL
			AND ec.deleted_at IS NULL
		ORDER BY ei.required_level ASC
	`

	var items []*domain.EquipmentItem
	err := r.db.Select(&items, query, slug, artifact)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *EquipmentItemRepository) FindByID(id uuid.UUID) (*domain.EquipmentItem, error) {
	query := `
		SELECT id, created_at, deleted_at, name, slug, attack, defense, hp,
			required_level, price, artifact, equipment_category_id, image
		FROM equipment_items
		WHERE id = $1 AND deleted_at IS NULL
	`

	item := &domain.EquipmentItem{}
	err := r.db.Get(item, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEquipmentItemNotFound
		}
		return nil, err
	}

	return item, nil
}

func (r *EquipmentItemRepository) FindByIDs(ids []uuid.UUID) ([]*domain.EquipmentItem, error) {
	query := `
		SELECT ei.id, ei.created_at, ei.deleted_at, ei.name, ei.slug, ei.attack, ei.defense, ei.hp,
			required_level, ei.price, ei.artifact, ei.equipment_category_id, ei.image, ec.type as equipment_type
		FROM equipment_items ei
		INNER JOIN equipment_categories ec 
		    ON ei.equipment_category_id = ec.id
		WHERE ei.id = ANY($1) 
		  AND ei.deleted_at IS NULL
	`

	items := []*domain.EquipmentItem{}
	if err := r.db.Select(&items, query, pq.Array(ids)); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *EquipmentItemRepository) FindBySlug(slug string) (*domain.EquipmentItem, error) {
	query := `
		SELECT id, created_at, deleted_at, name, slug, attack, defense, hp,
			required_level, price, artifact, equipment_category_id, image
		FROM equipment_items
		WHERE slug = $1 AND deleted_at IS NULL
	`

	item := &domain.EquipmentItem{}
	err := r.db.Get(item, query, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEquipmentItemNotFound
		}
		return nil, err
	}

	return item, nil
}

func (r *EquipmentItemRepository) Create(item *domain.EquipmentItem) error {
	query := `
		INSERT INTO equipment_items (name, slug, attack, defense, hp, required_level, price, artifact, equipment_category_id, image)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	err := r.db.QueryRow(query,
		item.Name, item.Slug, item.Attack, item.Defense, item.Hp,
		item.RequiredLevel, item.Price, item.Artifact, item.EquipmentCategoryID, item.Image,
	).Scan(&item.ID)
	if err != nil {
		return err
	}

	return nil
}
