package domain

import "github.com/google/uuid"

type Resource struct {
	Model
	Name           string    `db:"name" json:"name"`
	Slug           string    `db:"slug" json:"slug"`
	ToolCategoryID uuid.UUID `db:"tool_category_id" json:"tool_category_id"`
	ToolItemID     uuid.UUID `db:"tool_item_id" json:"tool_item_id"`
	LocationID     uuid.UUID `db:"location_id" json:"location_id"`
	Image          string    `db:"image" json:"image"`
}
