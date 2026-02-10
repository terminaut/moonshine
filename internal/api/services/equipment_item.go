package services

import (
	"context"
	"sort"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type EquipmentItemService struct {
	equipmentItemRepo *repository.EquipmentItemRepository
}

func NewEquipmentItemService(equipmentItemRepo *repository.EquipmentItemRepository) *EquipmentItemService {
	return &EquipmentItemService{
		equipmentItemRepo: equipmentItemRepo,
	}
}

func (s *EquipmentItemService) GetByCategorySlug(ctx context.Context, slug string, artifact bool) ([]*domain.EquipmentItem, error) {
	if slug == "ring" {
		rings, err := s.equipmentItemRepo.FindByCategorySlugAndArtifact("ring", artifact)
		if err != nil {
			return nil, err
		}
		necks, err := s.equipmentItemRepo.FindByCategorySlugAndArtifact("neck", artifact)
		if err != nil {
			return nil, err
		}
		out := make([]*domain.EquipmentItem, 0, len(rings)+len(necks))
		out = append(out, rings...)
		out = append(out, necks...)
		sort.Slice(out, func(i, j int) bool {
			if out[i].RequiredLevel != out[j].RequiredLevel {
				return out[i].RequiredLevel < out[j].RequiredLevel
			}
			return out[i].Name < out[j].Name
		})
		return out, nil
	}
	return s.equipmentItemRepo.FindByCategorySlugAndArtifact(slug, artifact)
}
