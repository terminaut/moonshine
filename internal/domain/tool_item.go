package domain

import "github.com/google/uuid"

type ToolItem struct {
	Model
	Name           string        `json:"name"`
	Price          uint          `json:"price"`
	RequiredSkill  uint          `json:"required_skill"`
	ToolCategoryID uuid.UUID     `json:"tool_category_id"`
	ToolCategory   *ToolCategory `json:"tool_category,omitempty"`
	Image          string        `json:"image"`
}
