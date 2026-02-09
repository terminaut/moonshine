package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	goredis "github.com/redis/go-redis/v9"

	"moonshine/internal/api/services"
	"moonshine/internal/api/ws"
	r "moonshine/internal/redis"
	"moonshine/internal/repository"
)

type HpWorker struct {
	healthRegenerationService *services.HealthRegenerationService
	userRepo                  *repository.UserRepository
	hub                       *ws.Hub
	rdb                       *goredis.Client
	ticker                    *time.Ticker
}

func NewHpWorker(db *sqlx.DB, rdb *goredis.Client, interval time.Duration) *HpWorker {
	userRepo := repository.NewUserRepository(db)
	healthRegenerationService := services.NewHealthRegenerationService(userRepo)

	return &HpWorker{
		healthRegenerationService: healthRegenerationService,
		userRepo:                  userRepo,
		hub:                       ws.GetHub(),
		rdb:                       rdb,
		ticker:                    time.NewTicker(interval),
	}
}

func (w *HpWorker) StartWorker(ctx context.Context) {
	defer w.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.ticker.C:
			w.regenerateHp()
		}
	}
}

func (w *HpWorker) regenerateHp() {
	regeneratedCount, err := w.healthRegenerationService.RegenerateAllUsers(1.0)
	if err != nil {
		fmt.Printf("[HpWorker] Error regenerating: %v\n", err)
		return
	}

	// Invalidate Redis cache for regenerated users
	userCache := r.UserCache(w.rdb)
	for _, update := range regeneratedCount {
		_ = userCache.Delete(context.Background(), update.UserID.String())
	}

	connectedUserIDs := w.hub.GetConnectedUserIDs()
	fmt.Printf("[HpWorker] Regenerated %d users, %d connected\n", len(regeneratedCount), len(connectedUserIDs))

	if len(connectedUserIDs) == 0 {
		return
	}

	updates, err := w.userRepo.GetHPForUsers(connectedUserIDs)
	if err != nil {
		fmt.Printf("[HpWorker] Error getting HP: %v\n", err)
		return
	}

	for _, update := range updates {
		err := w.hub.SendHPUpdate(update.UserID, update.CurrentHp, update.Hp)
		if err != nil {
			fmt.Printf("[HpWorker] Error sending HP update to %s: %v\n", update.UserID, err)
		} else {
			fmt.Printf("[HpWorker] Sent HP update to %s: %d/%d\n", update.UserID, update.CurrentHp, update.Hp)
		}
	}
}
