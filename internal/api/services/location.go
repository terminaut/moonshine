package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	goredis "github.com/redis/go-redis/v9"

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
	rdb          *goredis.Client
	locationRepo *repository.LocationRepository
	userRepo     *repository.UserRepository
	movingWorker MovingWorker
	graph        *LocationGraph
}

func NewLocationService(
	db *sqlx.DB,
	rdb *goredis.Client,
	locationRepo *repository.LocationRepository,
	userRepo *repository.UserRepository,
	movingWorker MovingWorker,
) (*LocationService, error) {
	graph, err := NewLocationGraph(locationRepo)
	if err != nil {
		return nil, err
	}

	return &LocationService{
		db:           db,
		rdb:          rdb,
		locationRepo: locationRepo,
		userRepo:     userRepo,
		movingWorker: movingWorker,
		graph:        graph,
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

	updateLocationQuery := `UPDATE users SET location_id = $1 WHERE id = $2`
	_, err = tx.Exec(updateLocationQuery, targetLocation.ID, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	userCache := r.UserCache(s.rdb)
	_ = userCache.Delete(ctx, userID.String())

	return nil
}

func (s *LocationService) FindShortestPath(fromSlug, toSlug string) ([]string, error) {
	return s.graph.FindShortestPath(fromSlug, toSlug)
}

func (s *LocationService) StartCellMovement(userID uuid.UUID, cellSlugs []string) error {
	return s.movingWorker.StartMovement(userID, cellSlugs)
}
