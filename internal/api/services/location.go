package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	r "moonshine/internal/redis"
	"moonshine/internal/repository"
)

var (
	ErrLocationNotConnected = errors.New("locations are not connected")
)

type MovingWorker interface {
	StartMovement(userID uuid.UUID, cellSlugs []string) error
}

type LocationService struct {
	db           *sqlx.DB
	locationRepo *repository.LocationRepository
	userRepo     *repository.UserRepository
	movingWorker MovingWorker
	graph        *LocationGraph
	userCache    r.Cache[domain.User]
	cellsCache   r.Cache[[]domain.LocationCell]
}

func NewLocationService(
	db *sqlx.DB,
	locationRepo *repository.LocationRepository,
	userRepo *repository.UserRepository,
	movingWorker MovingWorker,
	userCache r.Cache[domain.User],
	cellsCache r.Cache[[]domain.LocationCell],
) (*LocationService, error) {
	graph, err := NewLocationGraph(locationRepo)
	if err != nil {
		return nil, err
	}

	return &LocationService{
		db:           db,
		locationRepo: locationRepo,
		userRepo:     userRepo,
		movingWorker: movingWorker,
		graph:        graph,
		userCache:    userCache,
		cellsCache:   cellsCache,
	}, nil
}

func (s *LocationService) MoveToLocation(ctx context.Context, userID uuid.UUID, targetLocationSlug string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return repository.ErrUserNotFound
	}

	var targetLocation *domain.Location
	if targetLocationSlug == domain.WaywardPinesSlug {
		defaultOutDoorLocation, err := s.locationRepo.DefaultOutdoorLocation()
		if err != nil {
			return repository.ErrLocationNotFound
		}
		targetLocation = defaultOutDoorLocation
	} else {
		targetLocation, err = s.locationRepo.FindBySlug(targetLocationSlug)
		if err != nil {
			return repository.ErrLocationNotFound
		}
	}

	if user.LocationID == targetLocation.ID {
		return nil
	}

	if err = s.userRepo.UpdateLocationIDWithExt(tx, userID, targetLocation.ID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if s.userCache != nil {
		_ = s.userCache.Delete(ctx, userID.String())
	}

	return nil
}

func (s *LocationService) FindShortestPath(fromSlug, toSlug string) ([]string, error) {
	return s.graph.FindShortestPath(fromSlug, toSlug)
}

func (s *LocationService) StartCellMovement(userID uuid.UUID, cellSlugs []string) error {
	return s.movingWorker.StartMovement(userID, cellSlugs)
}

func (s *LocationService) FetchCells(ctx context.Context, locationID uuid.UUID) ([]domain.LocationCell, error) {
	cacheKey := locationID.String()

	if s.cellsCache != nil {
		cached, err := s.cellsCache.Get(ctx, cacheKey)
		if err == nil && cached != nil && *cached != nil && len(*cached) > 0 {
			return *cached, nil
		}
	}

	cells, err := s.locationRepo.FindCellsByLocationID(locationID)
	if err != nil {
		return nil, err
	}

	cellsList := make([]domain.LocationCell, len(cells))
	for i, cell := range cells {
		image := ""
		if cell.Image != nil {
			image = *cell.Image
		}
		cellsList[i] = domain.LocationCell{
			ID:       cell.ID.String(),
			Slug:     cell.Slug,
			Name:     cell.Name,
			Image:    image,
			Inactive: cell.Inactive,
		}
	}

	if s.cellsCache != nil && len(cellsList) > 0 {
		_ = s.cellsCache.Set(ctx, cacheKey, &cellsList)
	}

	return cellsList, nil
}
