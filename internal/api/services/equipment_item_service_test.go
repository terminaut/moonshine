package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/repository"
	"moonshine/internal/testutil"
)

func TestEquipmentItemService_GetByCategorySlug(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	service := NewEquipmentItemService(repository.NewEquipmentItemRepository(testDB))

	t.Run("weapon category", func(t *testing.T) {
		items, err := service.GetByCategorySlug(ctx, "weapon", false)
		require.NoError(t, err)
		assert.NotNil(t, items)
	})

	t.Run("ring merges ring and neck", func(t *testing.T) {
		items, err := service.GetByCategorySlug(ctx, "ring", false)
		require.NoError(t, err)
		assert.NotNil(t, items)
	})
}
