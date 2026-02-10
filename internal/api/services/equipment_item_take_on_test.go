package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

func setupTestData(db *sqlx.DB) (*domain.User, *domain.EquipmentItem, uuid.UUID, error) {
	location := &domain.Location{
		Name:     "Test Location",
		Slug:     fmt.Sprintf("test_location_%d", time.Now().UnixNano()),
		Cell:     false,
		Inactive: false,
	}
	locationRepo := repository.NewLocationRepository(db)
	err := locationRepo.Create(location)
	if err != nil {
		return nil, nil, uuid.Nil, fmt.Errorf("failed to create location: %w", err)
	}

	categoryQuery := `INSERT INTO equipment_categories (name, type) VALUES ($1, $2::equipment_category_type) RETURNING id, created_at, updated_at`
	category := &domain.EquipmentCategory{
		Name: "Weapon",
		Type: "weapon",
	}
	err = db.QueryRow(categoryQuery, category.Name, category.Type).Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		return nil, nil, uuid.Nil, fmt.Errorf("failed to create category: %w", err)
	}

	item := &domain.EquipmentItem{
		Name:                "Test Sword",
		Slug:                "test-sword",
		Attack:              10,
		Defense:             5,
		Hp:                  20,
		RequiredLevel:       1,
		Price:               100,
		EquipmentCategoryID: category.ID,
	}
	itemRepo := repository.NewEquipmentItemRepository(db)
	err = itemRepo.Create(item)
	if err != nil {
		return nil, nil, uuid.Nil, fmt.Errorf("failed to create item: %w", err)
	}

	user := &domain.User{
		Username:   fmt.Sprintf("testuser%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("test%d@example.com", time.Now().UnixNano()),
		Password:   "password",
		LocationID: location.ID,
		Attack:     1,
		Defense:    1,
		Hp:         20,
		CurrentHp:  20,
		Level:      5,
	}
	userRepo := repository.NewUserRepository(db)
	err = userRepo.Create(user)
	if err != nil {
		return nil, nil, uuid.Nil, fmt.Errorf("failed to create user: %w", err)
	}

	inventory := &domain.Inventory{
		UserID:          user.ID,
		EquipmentItemID: item.ID,
	}
	inventoryRepo := repository.NewInventoryRepository(db)
	err = inventoryRepo.Create(inventory)
	if err != nil {
		return nil, nil, uuid.Nil, fmt.Errorf("failed to add item to inventory: %w", err)
	}

	return user, item, category.ID, nil
}

func TestEquipmentItemTakeOnService_TakeOnEquipmentItem(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	db := testDB.DB()
	ctx := context.Background()

	user, item, categoryID, err := setupTestData(db)
	require.NoError(t, err)

	equipmentItemRepo := repository.NewEquipmentItemRepository(db)
	inventoryRepo := repository.NewInventoryRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewEquipmentItemTakeOnService(db, equipmentItemRepo, inventoryRepo, userRepo)

	t.Run("successfully equip item", func(t *testing.T) {
		err := service.TakeOnEquipmentItem(ctx, user.ID, item.ID)
		require.NoError(t, err)

		var equippedItemID uuid.UUID
		query := `SELECT weapon_equipment_item_id FROM users WHERE id = $1`
		err = db.Get(&equippedItemID, query, user.ID)
		require.NoError(t, err)
		assert.Equal(t, item.ID, equippedItemID)

		type stats struct {
			Attack  uint `db:"attack"`
			Defense uint `db:"defense"`
			Hp      uint `db:"hp"`
		}
		var userStats stats
		statsQuery := `SELECT attack, defense, hp FROM users WHERE id = $1`
		err = db.Get(&userStats, statsQuery, user.ID)
		require.NoError(t, err)

		assert.Equal(t, uint(11), userStats.Attack)
		assert.Equal(t, uint(6), userStats.Defense)
		assert.Equal(t, uint(40), userStats.Hp)
	})

	t.Run("item not in inventory", func(t *testing.T) {
		newItem := &domain.EquipmentItem{
			Name:                "New Sword",
			Slug:                fmt.Sprintf("new-sword-%d", time.Now().UnixNano()),
			Attack:              10,
			Defense:             5,
			Hp:                  20,
			RequiredLevel:       1,
			Price:               100,
			EquipmentCategoryID: categoryID,
		}
		err := equipmentItemRepo.Create(newItem)
		require.NoError(t, err)

		err = service.TakeOnEquipmentItem(ctx, user.ID, newItem.ID)
		assert.ErrorIs(t, err, ErrItemNotInInventory)
	})

	t.Run("insufficient level", func(t *testing.T) {
		highLevelItem := &domain.EquipmentItem{
			Name:                "High Level Sword",
			Slug:                fmt.Sprintf("high-level-sword-%d", time.Now().UnixNano()),
			Attack:              10,
			Defense:             5,
			Hp:                  20,
			RequiredLevel:       10,
			Price:               100,
			EquipmentCategoryID: categoryID,
		}
		err := equipmentItemRepo.Create(highLevelItem)
		require.NoError(t, err)

		inventory := &domain.Inventory{
			UserID:          user.ID,
			EquipmentItemID: highLevelItem.ID,
		}
		err = inventoryRepo.Create(inventory)
		require.NoError(t, err)

		err = service.TakeOnEquipmentItem(ctx, user.ID, highLevelItem.ID)
		assert.ErrorIs(t, err, ErrInsufficientLevel)
	})

	t.Run("replace existing equipment", func(t *testing.T) {
		newItem2 := &domain.EquipmentItem{
			Name:                "New Sword 2",
			Slug:                fmt.Sprintf("new-sword-2-%d", time.Now().UnixNano()),
			Attack:              15,
			Defense:             8,
			Hp:                  25,
			RequiredLevel:       1,
			Price:               100,
			EquipmentCategoryID: categoryID,
		}
		err := equipmentItemRepo.Create(newItem2)
		require.NoError(t, err)

		inventory2 := &domain.Inventory{
			UserID:          user.ID,
			EquipmentItemID: newItem2.ID,
		}
		err = inventoryRepo.Create(inventory2)
		require.NoError(t, err)

		err = service.TakeOnEquipmentItem(ctx, user.ID, newItem2.ID)
		require.NoError(t, err)

		var equippedItemID uuid.UUID
		query := `SELECT weapon_equipment_item_id FROM users WHERE id = $1`
		err = db.Get(&equippedItemID, query, user.ID)
		require.NoError(t, err)
		assert.Equal(t, newItem2.ID, equippedItemID)

		type stats struct {
			Attack  uint `db:"attack"`
			Defense uint `db:"defense"`
			Hp      uint `db:"hp"`
		}
		var userStats stats
		statsQuery := `SELECT attack, defense, hp FROM users WHERE id = $1`
		err = db.Get(&userStats, statsQuery, user.ID)
		require.NoError(t, err)

		assert.Equal(t, uint(16), userStats.Attack)
		assert.Equal(t, uint(9), userStats.Defense)
		assert.Equal(t, uint(45), userStats.Hp)

		var inventoryCount int
		inventoryCountQuery := `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND equipment_item_id = $2`
		err = db.Get(&inventoryCount, inventoryCountQuery, user.ID, item.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, inventoryCount)
	})
}
