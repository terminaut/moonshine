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

func TestInventoryService_GetUserInventory(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	service := NewInventoryService(repository.NewInventoryRepository(testDB))

	t.Run("empty inventory", func(t *testing.T) {
		user := setupUserServiceTestData(t)

		items, err := service.GetUserInventory(ctx, user.ID)
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("with items", func(t *testing.T) {
		user, item := setupBuyTestData(t)
		_, err := testDB.Exec(`INSERT INTO inventory (id, user_id, equipment_item_id) VALUES ($1, $2, $3)`, uuid.New(), user.ID, item.ID)
		require.NoError(t, err)

		items, err := service.GetUserInventory(ctx, user.ID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, item.Name, items[0].Name)
	})
}
