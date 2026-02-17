package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/repository"
	"moonshine/internal/testutil"
)

func TestEquipmentItemSellService_SellEquipmentItem(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		user, item := setupBuyTestData(t)
		initialGold := user.Gold

		_, err := testDB.Exec(`INSERT INTO inventory (id, user_id, equipment_item_id) VALUES ($1, $2, $3)`, uuid.New(), user.ID, item.ID)
		require.NoError(t, err)

		service := NewEquipmentItemSellService(
			testDB,
			repository.NewEquipmentItemRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err = service.SellEquipmentItem(ctx, user.ID, item.Slug)
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, initialGold+item.Price, userAfter.Gold)

		var count int
		err = testDB.Get(&count, `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND equipment_item_id = $2`, user.ID, item.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("item not owned", func(t *testing.T) {
		user, item := setupBuyTestData(t)
		service := NewEquipmentItemSellService(
			testDB,
			repository.NewEquipmentItemRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err := service.SellEquipmentItem(ctx, user.ID, item.Slug)
		assert.ErrorIs(t, err, ErrItemNotOwned)
	})

	t.Run("item not found", func(t *testing.T) {
		user, _ := setupBuyTestData(t)
		service := NewEquipmentItemSellService(
			testDB,
			repository.NewEquipmentItemRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)

		err := service.SellEquipmentItem(ctx, user.ID, "nonexistent-slug")
		assert.ErrorIs(t, err, ErrEquipmentItemNotFound)
	})
}
