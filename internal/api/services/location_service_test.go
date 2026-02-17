package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/domain"
	r "moonshine/internal/redis"
	"moonshine/internal/repository"
	"moonshine/internal/testutil"
)

func TestLocationService_MoveToLocation(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()
	ts := time.Now().UnixNano()

	locationRepo := repository.NewLocationRepository(testDB)
	userRepo := repository.NewUserRepository(testDB)

	loc1 := &domain.Location{Name: fmt.Sprintf("MoveLoc1 %d", ts), Slug: fmt.Sprintf("move-loc1-%d", ts)}
	require.NoError(t, locationRepo.Create(loc1))
	loc2 := &domain.Location{Name: fmt.Sprintf("MoveLoc2 %d", ts), Slug: fmt.Sprintf("move-loc2-%d", ts)}
	require.NoError(t, locationRepo.Create(loc2))

	user := &domain.User{
		Username:   fmt.Sprintf("mover%d", ts%1000000),
		Email:      fmt.Sprintf("mover%d@test.com", ts),
		Password:   "pass",
		LocationID: loc1.ID,
		Hp:         100,
		CurrentHp:  100,
		Level:      1,
	}
	require.NoError(t, userRepo.Create(user))

	service := &LocationService{
		db:           testDB,
		locationRepo: locationRepo,
		userRepo:     userRepo,
		movingWorker: noopMovingWorker{},
		userCache:    r.NewJSONCache[domain.User](nil, "user", 0),
	}

	t.Run("move to new location", func(t *testing.T) {
		err := service.MoveToLocation(ctx, user.ID, loc2.Slug)
		require.NoError(t, err)

		updated, err := userRepo.FindByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, loc2.ID, updated.LocationID)
	})

	t.Run("move to same location is noop", func(t *testing.T) {
		err := service.MoveToLocation(ctx, user.ID, loc2.Slug)
		require.NoError(t, err)
	})

	t.Run("move to nonexistent location", func(t *testing.T) {
		err := service.MoveToLocation(ctx, user.ID, "nonexistent-slug")
		assert.ErrorIs(t, err, repository.ErrLocationNotFound)
	})
}

func TestLocationService_FetchCells(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	locationRepo := repository.NewLocationRepository(testDB)

	service := &LocationService{
		db:           testDB,
		locationRepo: locationRepo,
		cellsCache:   r.NewJSONCache[[]domain.LocationCell](nil, "cells", 0),
	}

	t.Run("returns cells", func(t *testing.T) {
		cells, err := service.FetchCells(ctx, domain.Model{}.ID)
		require.NoError(t, err)
		assert.NotNil(t, cells)
	})
}
