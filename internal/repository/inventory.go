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
		INSERT INTO inventory (user_id, equipment_item_id)
		VALUES ($1, $2)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(query,
		inventory.UserID,
		inventory.EquipmentItemID,
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
