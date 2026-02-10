package worker

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"moonshine/internal/domain"
	r "moonshine/internal/redis"
	"moonshine/internal/repository"
)

type CellsMovingWorker struct {
	locationRepo *repository.LocationRepository
	userRepo     *repository.UserRepository
	userCache    r.Cache[domain.User]
	interval     time.Duration
	mu           sync.Mutex
	activeUsers  map[uuid.UUID]context.CancelFunc
}

func NewCellsMovingWorker(
	locationRepo *repository.LocationRepository,
	userRepo *repository.UserRepository,
	rdb *goredis.Client,
	interval time.Duration,
) *CellsMovingWorker {
	return &CellsMovingWorker{
		locationRepo: locationRepo,
		userRepo:     userRepo,
		userCache:    r.NewJSONCache[domain.User](rdb, "user", 5*time.Second),
		interval:     interval,
		activeUsers:  make(map[uuid.UUID]context.CancelFunc),
	}
}

func (w *CellsMovingWorker) StartMovement(userID uuid.UUID, cellSlugs []string) error {
	w.mu.Lock()
	if cancel, exists := w.activeUsers[userID]; exists {
		cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.activeUsers[userID] = cancel
	w.mu.Unlock()

	go func() {
		defer func() {
			w.mu.Lock()
			delete(w.activeUsers, userID)
			w.mu.Unlock()
		}()

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for _, cellSlug := range cellSlugs {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				location, _ := w.locationRepo.FindBySlug(cellSlug)
				if location == nil {
					continue
				}

				err := w.userRepo.UpdateLocationID(userID, location.ID)
				if err != nil {
					return
				}

				_ = w.userCache.Delete(context.Background(), userID.String())
			}
		}
	}()

	return nil
}
