package dto

import (
	"time"

	"moonshine/internal/domain"
)

type ToolItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Price     int       `json:"price"`
	Image     string    `json:"image"`
	CreatedAt time.Time `json:"createdAt"`
}

func ToolItemFromDomain(item *domain.ToolItem) *ToolItem {
	if item == nil {
		return nil
	}

	return &ToolItem{
		ID:        item.ID.String(),
		Name:      item.Name,
		Slug:      item.Slug,
		Price:     int(item.Price),
		Image:     item.Image,
		CreatedAt: item.CreatedAt,
	}
}

func ToolItemsFromDomain(items []*domain.ToolItem) []*ToolItem {
	result := make([]*ToolItem, len(items))
	for i, item := range items {
		result[i] = ToolItemFromDomain(item)
	}
	return result
}
