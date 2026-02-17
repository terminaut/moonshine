package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
	"moonshine/internal/testutil"
)

func setupUserServiceTestData(t *testing.T) *domain.User {
	t.Helper()
	ts := time.Now().UnixNano()

	locationRepo := repository.NewLocationRepository(testDB)
	location := &domain.Location{
		Name: fmt.Sprintf("UserSvcLoc %d", ts),
		Slug: fmt.Sprintf("user-svc-loc-%d", ts),
	}
	require.NoError(t, locationRepo.Create(location))

	userRepo := repository.NewUserRepository(testDB)
	user := &domain.User{
		Username:   fmt.Sprintf("usvc%d", ts%1000000),
		Email:      fmt.Sprintf("usvc%d@test.com", ts),
		Password:   "pass",
		LocationID: location.ID,
		Hp:         100,
		CurrentHp:  100,
		Level:      1,
	}
	require.NoError(t, userRepo.Create(user))
	return user
}

func TestUserService_GetCurrentUser(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	service := NewUserService(
		repository.NewUserRepository(testDB),
		repository.NewAvatarRepository(testDB),
		repository.NewLocationRepository(testDB),
		nil,
	)

	t.Run("success", func(t *testing.T) {
		user := setupUserServiceTestData(t)

		result, err := service.GetCurrentUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.ID)
		assert.Equal(t, user.Username, result.Username)
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := service.GetCurrentUser(ctx, uuid.New())
		assert.ErrorIs(t, err, repository.ErrUserNotFound)
	})
}

func TestUserService_GetCurrentUserWithRelations(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	service := NewUserService(
		repository.NewUserRepository(testDB),
		repository.NewAvatarRepository(testDB),
		repository.NewLocationRepository(testDB),
		nil,
	)

	t.Run("success", func(t *testing.T) {
		user := setupUserServiceTestData(t)

		resultUser, location, inFight, err := service.GetCurrentUserWithRelations(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, resultUser.ID)
		assert.NotNil(t, location)
		assert.Equal(t, user.LocationID, location.ID)
		assert.False(t, inFight)
	})

	t.Run("user not found", func(t *testing.T) {
		_, _, _, err := service.GetCurrentUserWithRelations(ctx, uuid.New())
		assert.ErrorIs(t, err, repository.ErrUserNotFound)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	service := NewUserService(
		repository.NewUserRepository(testDB),
		repository.NewAvatarRepository(testDB),
		repository.NewLocationRepository(testDB),
		nil,
	)

	t.Run("update with nil avatar", func(t *testing.T) {
		user := setupUserServiceTestData(t)

		result, err := service.UpdateUser(ctx, user.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.ID)
	})

	t.Run("avatar not found", func(t *testing.T) {
		user := setupUserServiceTestData(t)
		nonExistentID := uuid.New()

		_, err := service.UpdateUser(ctx, user.ID, &nonExistentID)
		assert.ErrorIs(t, err, repository.ErrAvatarNotFound)
	})
}
