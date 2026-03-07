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
)

func TestResourceService_GatherResource_WrongLocation(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	db := testDB
	ctx := context.Background()

	locationRepo := repository.NewLocationRepository(db)
	resourceRepo := repository.NewResourceRepository(db)
	userRepo := repository.NewUserRepository(db)
	inventoryRepo := repository.NewInventoryRepository(db)

	ts := time.Now().UnixNano()

	userLocation := &domain.Location{
		Name: "User Location",
		Slug: fmt.Sprintf("user_loc_%d", ts),
	}
	require.NoError(t, locationRepo.Create(userLocation))

	resourceLocation := &domain.Location{
		Name: "Resource Location",
		Slug: fmt.Sprintf("res_loc_%d", ts),
	}
	require.NoError(t, locationRepo.Create(resourceLocation))

	var toolCategoryID uuid.UUID
	err := db.QueryRow(
		`INSERT INTO tool_categories (id, name, type) VALUES ($1, $2, $3) RETURNING id`,
		uuid.New(), fmt.Sprintf("TestCat_%d", ts), fmt.Sprintf("test_%d", ts),
	).Scan(&toolCategoryID)
	require.NoError(t, err)

	var toolItemID uuid.UUID
	err = db.QueryRow(
		`INSERT INTO tool_items (id, name, slug, price, tool_category_id) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		uuid.New(), fmt.Sprintf("TestTool_%d", ts), fmt.Sprintf("test_tool_%d", ts), 10, toolCategoryID,
	).Scan(&toolItemID)
	require.NoError(t, err)

	resourceSlug := fmt.Sprintf("test_res_%d", ts)
	var resourceID uuid.UUID
	err = db.QueryRow(
		`INSERT INTO resources (id, name, slug, tool_category_id, tool_item_id) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		uuid.New(), "Test Resource", resourceSlug, toolCategoryID, toolItemID,
	).Scan(&resourceID)
	require.NoError(t, err)

	_, err = db.Exec(
		`INSERT INTO location_resources (id, location_id, resource_id) VALUES ($1, $2, $3)`,
		uuid.New(), resourceLocation.ID, resourceID,
	)
	require.NoError(t, err)

	user := &domain.User{
		Username:   fmt.Sprintf("testuser_%d", ts),
		Email:      fmt.Sprintf("test_%d@example.com", ts),
		Password:   "password",
		LocationID: userLocation.ID,
		Attack:     1,
		Defense:    1,
		Hp:        20,
		CurrentHp: 20,
		Level:     1,
	}
	require.NoError(t, userRepo.Create(user))

	service := NewResourceService(locationRepo, resourceRepo, userRepo, inventoryRepo)

	t.Run("user at wrong location returns error", func(t *testing.T) {
		err := service.GatherResource(ctx, user.ID, resourceSlug)
		assert.ErrorIs(t, err, repository.ErrResourceNotFound)
	})

	t.Run("user at correct location succeeds", func(t *testing.T) {
		_, err := db.Exec(
			`UPDATE users SET location_id = $1 WHERE id = $2`,
			resourceLocation.ID, user.ID,
		)
		require.NoError(t, err)

		err = service.GatherResource(ctx, user.ID, resourceSlug)
		assert.NoError(t, err)
	})
}
