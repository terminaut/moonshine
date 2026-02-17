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

func setupBuyTestData(t *testing.T) (*domain.User, *domain.EquipmentItem) {
	t.Helper()
	testutil.RequireDB(t, testDB)
	ts := time.Now().UnixNano()

	locationRepo := repository.NewLocationRepository(testDB)
	location := &domain.Location{
		Name: fmt.Sprintf("BuyLoc %d", ts),
		Slug: fmt.Sprintf("buy-loc-%d", ts),
	}
	require.NoError(t, locationRepo.Create(location))

	userRepo := repository.NewUserRepository(testDB)
	user := &domain.User{
		Username:   fmt.Sprintf("buyer%d", ts%1000000),
		Email:      fmt.Sprintf("buyer%d@test.com", ts),
		Password:   "pass",
		LocationID: location.ID,
		Gold:       500,
		Hp:         100,
		CurrentHp:  100,
		Level:      1,
	}
	require.NoError(t, userRepo.Create(user))

	var categoryID uuid.UUID
	err := testDB.Get(&categoryID, `SELECT id FROM equipment_categories WHERE type = 'weapon' LIMIT 1`)
	if err != nil {
		_, err = testDB.Exec(`INSERT INTO equipment_categories (id, name, type, created_at) VALUES ($1, 'Weapon', 'weapon', NOW())`, uuid.New())
		require.NoError(t, err)
		err = testDB.Get(&categoryID, `SELECT id FROM equipment_categories WHERE type = 'weapon' LIMIT 1`)
	}
	require.NoError(t, err)

	equipmentItemRepo := repository.NewEquipmentItemRepository(testDB)
	item := &domain.EquipmentItem{
		Name:                fmt.Sprintf("Sword %d", ts),
		Slug:                fmt.Sprintf("sword-%d", ts),
		Price:               100,
		RequiredLevel:       1,
		EquipmentCategoryID: categoryID,
	}
	require.NoError(t, equipmentItemRepo.Create(item))

	return user, item
}

func TestEquipmentItemBuyService_BuyEquipmentItem(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		user, item := setupBuyTestData(t)
		service := NewEquipmentItemBuyService(
			testDB,
			repository.NewEquipmentItemRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err := service.BuyEquipmentItem(ctx, user.ID, item.Slug)
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, uint(400), userAfter.Gold)

		var count int
		err = testDB.Get(&count, `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND equipment_item_id = $2`, user.ID, item.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("insufficient gold", func(t *testing.T) {
		user, item := setupBuyTestData(t)
		testDB.Exec(`UPDATE users SET gold = 0 WHERE id = $1`, user.ID)

		service := NewEquipmentItemBuyService(
			testDB,
			repository.NewEquipmentItemRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err := service.BuyEquipmentItem(ctx, user.ID, item.Slug)
		assert.ErrorIs(t, err, ErrInsufficientGold)
	})

	t.Run("item not found", func(t *testing.T) {
		user, _ := setupBuyTestData(t)
		service := NewEquipmentItemBuyService(
			testDB,
			repository.NewEquipmentItemRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err := service.BuyEquipmentItem(ctx, user.ID, "nonexistent-item")
		assert.ErrorIs(t, err, ErrEquipmentItemNotFound)
	})

	t.Run("user not found", func(t *testing.T) {
		_, item := setupBuyTestData(t)
		service := NewEquipmentItemBuyService(
			testDB,
			repository.NewEquipmentItemRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err := service.BuyEquipmentItem(ctx, uuid.New(), item.Slug)
		assert.ErrorIs(t, err, repository.ErrUserNotFound)
	})
}
