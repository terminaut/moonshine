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

func setupPotionBuyTestData(t *testing.T) (*domain.User, *domain.Potion) {
	t.Helper()
	testutil.RequireDB(t, testDB)
	ts := time.Now().UnixNano()

	locationRepo := repository.NewLocationRepository(testDB)
	location := &domain.Location{
		Name: fmt.Sprintf("PotionBuyLoc %d", ts),
		Slug: fmt.Sprintf("potion-buy-loc-%d", ts),
	}
	require.NoError(t, locationRepo.Create(location))

	userRepo := repository.NewUserRepository(testDB)
	user := &domain.User{
		Username:   fmt.Sprintf("potion-buyer%d", ts%1000000),
		Email:      fmt.Sprintf("potion-buyer%d@test.com", ts),
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
		Name:  fmt.Sprintf("HP Potion %d", ts),
		Slug:  fmt.Sprintf("hp-potion-%d", ts),
		Stat:  "CURRENT_HP",
		Value: 50,
		Price: 100,
		Image: "potions/current_hp/big.png",
	}
	potion.ID = uuid.New()
	err := testDB.QueryRow(
		`INSERT INTO potions (id, name, slug, stat, value, price, image) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING created_at`,
		potion.ID, potion.Name, potion.Slug, potion.Stat, potion.Value, potion.Price, potion.Image,
	).Scan(&potion.CreatedAt)
	require.NoError(t, err)

	return user, potion
}

func TestPotionBuyService_BuyPotion(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		user, potion := setupPotionBuyTestData(t)
		initialGold := user.Gold

		service := NewPotionBuyService(
			testDB,
			repository.NewPotionRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err := service.BuyPotion(ctx, user.ID, potion.Slug)
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, initialGold-potion.Price, userAfter.Gold)

		var count int
		err = testDB.Get(&count, `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND potion_id = $2 AND deleted_at IS NULL`, user.ID, potion.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("insufficient gold", func(t *testing.T) {
		user, potion := setupPotionBuyTestData(t)

		_, err := testDB.Exec(`UPDATE users SET gold = 0 WHERE id = $1`, user.ID)
		require.NoError(t, err)

		service := NewPotionBuyService(
			testDB,
			repository.NewPotionRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err = service.BuyPotion(ctx, user.ID, potion.Slug)
		assert.ErrorIs(t, err, ErrInsufficientGold)
	})

	t.Run("potion not found", func(t *testing.T) {
		user, _ := setupPotionBuyTestData(t)

		service := NewPotionBuyService(
			testDB,
			repository.NewPotionRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err := service.BuyPotion(ctx, user.ID, "nonexistent-potion")
		assert.ErrorIs(t, err, ErrPotionNotFound)
	})
}
