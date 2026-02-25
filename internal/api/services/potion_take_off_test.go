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

func setupPotionTakeOffTestData(t *testing.T) (*domain.User, *domain.Potion) {
	t.Helper()
	testutil.RequireDB(t, testDB)
	ts := time.Now().UnixNano()

	locationRepo := repository.NewLocationRepository(testDB)
	location := &domain.Location{
		Name: fmt.Sprintf("PotionTakeOffLoc %d", ts),
		Slug: fmt.Sprintf("potion-takeoff-loc-%d", ts),
	}
	require.NoError(t, locationRepo.Create(location))

	userRepo := repository.NewUserRepository(testDB)
	user := &domain.User{
		Username:   fmt.Sprintf("potion-takeoff%d", ts%1000000),
		Email:      fmt.Sprintf("potion-takeoff%d@test.com", ts),
		Password:   "pass",
		LocationID: location.ID,
		Gold:       500,
		Hp:         100,
		CurrentHp:  100,
		Level:      1,
		Attack:     1,
		Defense:    1,
	}
	require.NoError(t, userRepo.Create(user))

	potion := &domain.Potion{
		Name:  fmt.Sprintf("TakeOff Potion %d", ts),
		Slug:  fmt.Sprintf("takeoff-potion-%d", ts),
		Stat:  "CURRENT_HP",
		Value: 50,
		Price: 100,
	}
	potion.ID = uuid.New()
	err := testDB.QueryRow(
		`INSERT INTO potions (id, name, slug, stat, value, price) VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at`,
		potion.ID, potion.Name, potion.Slug, potion.Stat, potion.Value, potion.Price,
	).Scan(&potion.CreatedAt)
	require.NoError(t, err)

	return user, potion
}

func equipPotionToSlot(t *testing.T, userID, potionID uuid.UUID, slot string) {
	t.Helper()
	col := slotToColumn(slot)
	_, err := testDB.Exec(fmt.Sprintf(`UPDATE users SET %s = $1 WHERE id = $2`, col), potionID, userID)
	require.NoError(t, err)
}

func TestPotionTakeOffService_TakeOffPotion(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	newService := func() *PotionTakeOffService {
		return NewPotionTakeOffService(
			testDB,
			repository.NewPotionRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)
	}

	t.Run("success", func(t *testing.T) {
		user, potion := setupPotionTakeOffTestData(t)
		equipPotionToSlot(t, user.ID, potion.ID, "potion1")

		err := newService().TakeOffPotion(ctx, user.ID, "potion1")
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		assert.Nil(t, userAfter.Potion1ID)

		var count int
		err = testDB.Get(&count, `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND potion_id = $2 AND deleted_at IS NULL`, user.ID, potion.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("take off slot 2", func(t *testing.T) {
		user, potion := setupPotionTakeOffTestData(t)
		equipPotionToSlot(t, user.ID, potion.ID, "potion2")

		err := newService().TakeOffPotion(ctx, user.ID, "potion2")
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		assert.Nil(t, userAfter.Potion2ID)
	})

	t.Run("no potion equipped", func(t *testing.T) {
		user, _ := setupPotionTakeOffTestData(t)

		err := newService().TakeOffPotion(ctx, user.ID, "potion1")
		assert.ErrorIs(t, err, ErrNoItemEquipped)
	})

	t.Run("invalid slot", func(t *testing.T) {
		user, _ := setupPotionTakeOffTestData(t)

		err := newService().TakeOffPotion(ctx, user.ID, "invalid")
		assert.ErrorIs(t, err, ErrInvalidEquipmentType)
	})
}
