package repository

import (
	"database/sql"
	"errors"

	"moonshine/internal/domain"

	"github.com/google/uuid"
)

var (
	ErrInventoryNotFound = errors.New("inventory item not found")
)

type dbInterface interface {
	Exec(query string, args ...any) (sql.Result, error)
	Select(dest any, query string, args ...any) error
	QueryRow(query string, args ...any) *sql.Row
}

type InventoryRepository struct {
	db dbInterface
}

func NewInventoryRepository(db dbInterface) *InventoryRepository {
	return &InventoryRepository{db: db}
}

func (r *InventoryRepository) Create(inventory *domain.Inventory) error {
	query := `
		INSERT INTO inventory (user_id, equipment_item_id, potion_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(query,
		inventory.UserID,
		inventory.EquipmentItemID,
		inventory.PotionID,
	).Scan(&inventory.ID, &inventory.CreatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *InventoryRepository) FindByUserID(userID uuid.UUID) ([]*domain.EquipmentItem, error) {
	query := `
		SELECT 
			ei.id, ei.created_at, ei.deleted_at, ei.name, ei.slug, ei.attack, ei.defense, ei.hp,
			ei.required_level, ei.price, ei.artifact, ei.equipment_category_id, ei.image
		FROM inventory i
		INNER JOIN equipment_items ei ON i.equipment_item_id = ei.id
		WHERE i.user_id = $1 
			AND i.deleted_at IS NULL
			AND ei.deleted_at IS NULL
		ORDER BY ei.name ASC
	`

	var items []*domain.EquipmentItem

	err := r.db.Select(&items, query, userID)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *InventoryRepository) FindPotionsByUserID(userID uuid.UUID) ([]*domain.Potion, error) {
	query := `
		SELECT
			p.id, p.created_at, p.deleted_at, p.name, p.slug, p.stat, p.value, p.price,
			COALESCE(p.image, '') as image
		FROM inventory i
		INNER JOIN potions p ON i.potion_id = p.id
		WHERE i.user_id = $1
			AND i.deleted_at IS NULL
			AND p.deleted_at IS NULL
		ORDER BY p.name ASC
	`

	var potions []*domain.Potion
	err := r.db.Select(&potions, query, userID)
	if err != nil {
		return nil, err
	}

	return potions, nil
}

func (r *InventoryRepository) HasPotion(userID, potionID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM inventory
			WHERE user_id = $1 AND potion_id = $2 AND deleted_at IS NULL
		)
	`
	var exists bool
	err := r.db.QueryRow(query, userID, potionID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *InventoryRepository) DeleteOnePotion(userID, potionID uuid.UUID) error {
	query := `
		DELETE FROM inventory
		WHERE id = (
			SELECT id
			FROM inventory
			WHERE user_id = $1 AND potion_id = $2 AND deleted_at IS NULL
			LIMIT 1
		)
	`
	_, err := r.db.Exec(query, userID, potionID)
	return err
}
