package domain

import "github.com/google/uuid"

type Inventory struct {
	Model
	UserID          uuid.UUID `db:"user_id"`
	EquipmentItemID uuid.UUID `db:"equipment_item_id"`
}
