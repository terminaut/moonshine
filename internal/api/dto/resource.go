package dto

import (
	"time"

	"moonshine/internal/domain"
)

type Resource struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Image     string    `json:"image"`
	CreatedAt time.Time `json:"createdAt"`
}

func ResourceFromDomain(resource *domain.Resource) *Resource {
	if resource == nil {
		return nil
	}

	return &Resource{
		ID:        resource.ID.String(),
		Name:      resource.Name,
		Slug:      resource.Slug,
		Image:     resource.Image,
		CreatedAt: resource.CreatedAt,
	}
}

func ResourcesFromDomain(resources []*domain.Resource) []*Resource {
	result := make([]*Resource, len(resources))
	for i, resource := range resources {
		result[i] = ResourceFromDomain(resource)
	}
	return result
}
