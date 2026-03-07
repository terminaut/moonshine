package services

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type ResourceService struct {
	locationRepo  *repository.LocationRepository
	resourceRepo  *repository.ResourceRepository
	userRepo      *repository.UserRepository
	inventoryRepo *repository.InventoryRepository
}

func NewResourceService(
	locationRepo *repository.LocationRepository,
	resourceRepo *repository.ResourceRepository,
	userRepo *repository.UserRepository,
	inventoryRepo *repository.InventoryRepository,
) *ResourceService {
	return &ResourceService{
		locationRepo:  locationRepo,
		resourceRepo:  resourceRepo,
		userRepo:      userRepo,
		inventoryRepo: inventoryRepo,
	}
}

func (s *ResourceService) GetResourcesByLocationSlug(locationSlug string) ([]*domain.Resource, error) {
	if locationSlug == "" {
		return nil, errors.New("location slug is required")
	}

	location, err := s.locationRepo.FindBySlug(locationSlug)
	if err != nil {
		return nil, err
	}

	resources, err := s.resourceRepo.FindResourcesByLocationID(location.ID)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func (s *ResourceService) GatherResource(_ context.Context, userID uuid.UUID, resourceSlug string) error {
	if resourceSlug == "" {
		return errors.New("resource slug is required")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	resource, err := s.resourceRepo.FindBySlug(resourceSlug)
	if err != nil {
		return err
	}

	if user.LocationID != resource.LocationID {
		return repository.ErrResourceNotFound
	}

	time.AfterFunc(10*time.Second, func() {
		inventory := &domain.Inventory{
			UserID:     user.ID,
			ResourceID: &resource.ID,
		}
		if err := s.inventoryRepo.Create(inventory); err != nil {
			log.Printf("[ResourceService] failed to add gathered resource to inventory: %v", err)
		}
	})

	return nil
}
