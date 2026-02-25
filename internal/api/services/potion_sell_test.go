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

func setupPotionSellTestData(t *testing.T) (*domain.User, *domain.Potion) {
	t.Helper()
	testutil.RequireDB(t, testDB)
	ts := time.Now().UnixNano()

	locationRepo := repository.NewLocationRepository(testDB)
	location := &domain.Location{
		Name: fmt.Sprintf("PotionSellLoc %d", ts),
		Slug: fmt.Sprintf("potion-sell-loc-%d", ts),
	}
	require.NoError(t, locationRepo.Create(location))

	userRepo := repository.NewUserRepository(testDB)
	user := &domain.User{
		Username:   fmt.Sprintf("potion-seller%d", ts%1000000),
		Email:      fmt.Sprintf("potion-seller%d@test.com", ts),
		Password:   "pass",
		LocationID: location.ID,
		Gold:       500,
		Hp:         100,
		CurrentHp:  100,
		Level:      1,
	}
	require.NoError(t, userRepo.Create(user))

	potion := &domain.Potion{
		Name:  fmt.Sprintf("Potion %d", ts),
		Slug:  fmt.Sprintf("potion-%d", ts),
		Stat:  "CURRENT_HP",
		Value: 50,
		Price: 100,
		Image: "potions/current_hp/1.png",
	}
	potion.ID = uuid.New()
	err := testDB.QueryRow(
		`INSERT INTO potions (id, name, slug, stat, value, price, image) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING created_at`,
		potion.ID, potion.Name, potion.Slug, potion.Stat, potion.Value, potion.Price, potion.Image,
	).Scan(&potion.CreatedAt)
	require.NoError(t, err)

	return user, potion
}

func TestPotionSellService_SellPotion(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		user, potion := setupPotionSellTestData(t)
		initialGold := user.Gold

		_, err := testDB.Exec(
			`INSERT INTO inventory (id, user_id, potion_id) VALUES ($1, $2, $3)`,
			uuid.New(), user.ID, potion.ID,
		)
		require.NoError(t, err)

		service := NewPotionSellService(
			testDB,
			repository.NewPotionRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err = service.SellPotion(ctx, user.ID, potion.Slug)
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, initialGold+(potion.Price*9/10), userAfter.Gold)

		var count int
		err = testDB.Get(&count, `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND potion_id = $2`, user.ID, potion.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("potion not owned", func(t *testing.T) {
		user, potion := setupPotionSellTestData(t)
		service := NewPotionSellService(
			testDB,
			repository.NewPotionRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err := service.SellPotion(ctx, user.ID, potion.Slug)
		assert.ErrorIs(t, err, ErrItemNotOwned)
	})

	t.Run("potion not found", func(t *testing.T) {
		user, _ := setupPotionSellTestData(t)
		service := NewPotionSellService(
			testDB,
			repository.NewPotionRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err := service.SellPotion(ctx, user.ID, "nonexistent-potion")
		assert.ErrorIs(t, err, ErrPotionNotFound)
	})
}
