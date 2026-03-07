package domain

import "github.com/google/uuid"

type ToolItem struct {
	Model
	Name           string        `db:"name" json:"name"`
	Slug           string        `db:"slug" json:"slug"`
	Price          uint          `db:"price" json:"price"`
	ToolCategoryID uuid.UUID     `db:"tool_category_id" json:"tool_category_id"`
	ToolCategory   *ToolCategory `json:"tool_category,omitempty"`
	Image          string        `db:"image" json:"image"`
}
