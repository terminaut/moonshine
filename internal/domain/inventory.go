package domain

import "github.com/google/uuid"

type Inventory struct {
	Model
	UserID          uuid.UUID  `db:"user_id"`
	EquipmentItemID *uuid.UUID `db:"equipment_item_id"`
	PotionID        *uuid.UUID `db:"potion_id"`
	ToolItemID      *uuid.UUID `db:"tool_item_id"`
	ResourceID      *uuid.UUID `db:"resource_id"`
}
