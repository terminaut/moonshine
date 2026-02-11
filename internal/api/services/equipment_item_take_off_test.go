package services

import (
	"context"
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

func setupTestDataForTakeOff(db *sqlx.DB) (*domain.User, *domain.EquipmentItem, uuid.UUID, error) {
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

	categoryQuery := `INSERT INTO equipment_categories (name, type) VALUES ($1, $2::equipment_category_type) RETURNING id, created_at`
	category := &domain.EquipmentCategory{
		Name: "Weapon",
		Type: "weapon",
	}
	err = db.QueryRow(categoryQuery, category.Name, category.Type).Scan(&category.ID, &category.CreatedAt)
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

	userQuery := `INSERT INTO users (username, email, password, location_id, attack, defense, hp, current_hp, level, weapon_equipment_item_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`
	ts := time.Now().UnixNano()
	username := fmt.Sprintf("testuser%d", ts)
	user := &domain.User{
		Username:              username,
		Email:                 fmt.Sprintf("test%d@example.com", ts),
		Password:              "password",
		LocationID:            location.ID,
		Attack:                11,
		Defense:               6,
		Hp:                    40,
		CurrentHp:             40,
		Level:                 5,
		WeaponEquipmentItemID: &item.ID,
	}
	err = db.QueryRow(userQuery, user.Username, user.Email, user.Password, user.LocationID,
		user.Attack, user.Defense, user.Hp, user.CurrentHp, user.Level, user.WeaponEquipmentItemID,
	).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return nil, nil, uuid.Nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, item, category.ID, nil
}

func TestEquipmentItemTakeOffService_TakeOffEquipmentItem(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	db := testDB.DB()
	ctx := context.Background()

	user, item, _, err := setupTestDataForTakeOff(db)
	require.NoError(t, err)

	equipmentItemRepo := repository.NewEquipmentItemRepository(db)
	inventoryRepo := repository.NewInventoryRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewEquipmentItemTakeOffService(db, equipmentItemRepo, inventoryRepo, userRepo)

	t.Run("successfully unequip item", func(t *testing.T) {
		err := service.TakeOffEquipmentItem(ctx, user.ID, "weapon")
		require.NoError(t, err)

		var equippedItemID *uuid.UUID
		query := `SELECT weapon_equipment_item_id FROM users WHERE id = $1`
		err = db.Get(&equippedItemID, query, user.ID)
		require.NoError(t, err)
		assert.Nil(t, equippedItemID)

		type stats struct {
			Attack  uint `db:"attack"`
			Defense uint `db:"defense"`
			Hp      uint `db:"hp"`
		}
		var userStats stats
		statsQuery := `SELECT attack, defense, hp FROM users WHERE id = $1`
		err = db.Get(&userStats, statsQuery, user.ID)
		require.NoError(t, err)

		assert.Equal(t, uint(1), userStats.Attack)
		assert.Equal(t, uint(1), userStats.Defense)
		assert.Equal(t, uint(20), userStats.Hp)

		var inventoryCount int
		inventoryCountQuery := `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND equipment_item_id = $2`
		err = db.Get(&inventoryCount, inventoryCountQuery, user.ID, item.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, inventoryCount)
	})

	t.Run("no item equipped in slot", func(t *testing.T) {
		newUserQuery := `INSERT INTO users (username, email, password, location_id, attack, defense, hp, current_hp, level)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id, created_at, updated_at`
		ts := time.Now().UnixNano()
		username := fmt.Sprintf("testuser%d", ts)
		var newUserID uuid.UUID
		err := db.QueryRow(newUserQuery, username, fmt.Sprintf("test%d@example.com", ts), "password", user.LocationID, 1, 1, 20, 20, 5).Scan(&newUserID, nil, nil)
		require.NoError(t, err)

		err = service.TakeOffEquipmentItem(ctx, newUserID, "weapon")
		assert.ErrorIs(t, err, ErrNoItemEquipped)
	})

	t.Run("invalid slot name", func(t *testing.T) {
		err := service.TakeOffEquipmentItem(ctx, user.ID, "invalid_slot")
		assert.ErrorIs(t, err, ErrInvalidEquipmentType)
	})

	t.Run("unequip one item with multiple equipped", func(t *testing.T) {
		locationID := uuid.New()
		locationQuery := `INSERT INTO locations (id, name, slug, cell, inactive) VALUES ($1, $2, $3, $4, $5)`
		_, err := db.Exec(locationQuery, locationID, "Test Location 2", "test_location_2", false, false)
		require.NoError(t, err)

		weaponCatID := uuid.New()
		categoryQuery := `INSERT INTO equipment_categories (id, name, type) VALUES ($1, $2, $3::equipment_category_type)`
		_, err = db.Exec(categoryQuery, weaponCatID, "Weapon", "weapon")
		require.NoError(t, err)

		chestCatID := uuid.New()
		_, err = db.Exec(categoryQuery, chestCatID, "Chest", "chest")
		require.NoError(t, err)

		weaponID := uuid.New()
		itemQuery := `INSERT INTO equipment_items (id, name, slug, attack, defense, hp, required_level, price, equipment_category_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		_, err = db.Exec(itemQuery, weaponID, "Test Weapon", "test-weapon", 10, 0, 0, 1, 100, weaponCatID)
		require.NoError(t, err)

		chestID := uuid.New()
		_, err = db.Exec(itemQuery, chestID, "Test Chest", "test-chest", 0, 15, 30, 1, 100, chestCatID)
		require.NoError(t, err)

		multiUserID := uuid.New()
		ts := time.Now().UnixNano()
		username := fmt.Sprintf("multiuser%d", ts)
		userQuery := `INSERT INTO users (id, username, email, password, location_id, attack, defense, hp, current_hp, level, weapon_equipment_item_id, chest_equipment_item_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
		_, err = db.Exec(userQuery, multiUserID, username, fmt.Sprintf("multi%d@example.com", ts), "password", locationID, 11, 16, 50, 50, 5, weaponID, chestID)
		require.NoError(t, err)

		err = service.TakeOffEquipmentItem(ctx, multiUserID, "weapon")
		require.NoError(t, err)

		type stats struct {
			Attack  uint `db:"attack"`
			Defense uint `db:"defense"`
			Hp      uint `db:"hp"`
		}
		var userStats stats
		statsQuery := `SELECT attack, defense, hp FROM users WHERE id = $1`
		err = db.Get(&userStats, statsQuery, multiUserID)
		require.NoError(t, err)

		assert.Equal(t, uint(1), userStats.Attack)
		assert.Equal(t, uint(16), userStats.Defense)
		assert.Equal(t, uint(50), userStats.Hp)

		var chestEquippedID uuid.UUID
		chestQuery := `SELECT chest_equipment_item_id FROM users WHERE id = $1`
		err = db.Get(&chestEquippedID, chestQuery, multiUserID)
		require.NoError(t, err)
		assert.Equal(t, chestID, chestEquippedID)
	})
}
