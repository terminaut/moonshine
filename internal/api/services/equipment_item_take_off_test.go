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
		Slug:                fmt.Sprintf("test-sword-%d", time.Now().UnixNano()),
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

	ts := time.Now().UnixNano()
	username := fmt.Sprintf("testuser%d", ts)
	user := &domain.User{
		Username:              username,
		Name:                  username,
		Email:                 fmt.Sprintf("test%d@example.com", ts),
		Password:              "password",
		LocationID:            location.ID,
		Attack:                11,
		Defense:               6,
		Hp:                    40,
		CurrentHp:             40,
		Level:                 5,
		Exp:                   0,
		FreeStats:             15,
		Gold:                  100,
		WeaponEquipmentItemID: &item.ID,
	}
	userRepo := repository.NewUserRepository(db)
	err = userRepo.Create(user)
	if err != nil {
		return nil, nil, uuid.Nil, fmt.Errorf("failed to create user: %w", err)
	}

	updateQuery := `UPDATE users SET weapon_equipment_item_id = $1 WHERE id = $2`
	_, err = db.Exec(updateQuery, item.ID, user.ID)
	if err != nil {
		return nil, nil, uuid.Nil, fmt.Errorf("failed to equip weapon: %w", err)
	}

	return user, item, category.ID, nil
}

func TestEquipmentItemTakeOffService_TakeOffEquipmentItem(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	db := testDB
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
			Current int  `db:"current_hp"`
		}
		var userStats stats
		statsQuery := `SELECT attack, defense, hp, current_hp FROM users WHERE id = $1`
		err = db.Get(&userStats, statsQuery, user.ID)
		require.NoError(t, err)

		assert.Equal(t, uint(1), userStats.Attack)
		assert.Equal(t, uint(1), userStats.Defense)
		assert.Equal(t, uint(20), userStats.Hp)
		assert.Equal(t, 20, userStats.Current)

		var inventoryCount int
		inventoryCountQuery := `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND equipment_item_id = $2`
		err = db.Get(&inventoryCount, inventoryCountQuery, user.ID, item.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, inventoryCount)
	})

	t.Run("no item equipped in slot", func(t *testing.T) {
		ts := time.Now().UnixNano()
		newUser := &domain.User{
			Username:   fmt.Sprintf("u%d", ts%1000000),
			Email:      fmt.Sprintf("test%d@example.com", ts),
			Password:   "password",
			LocationID: user.LocationID,
			Attack:     1,
			Defense:    1,
			Hp:         20,
			CurrentHp:  20,
			Level:      5,
		}
		err := userRepo.Create(newUser)
		require.NoError(t, err)

		err = service.TakeOffEquipmentItem(ctx, newUser.ID, "weapon")
		assert.ErrorIs(t, err, ErrNoItemEquipped)
	})

	t.Run("invalid slot name", func(t *testing.T) {
		err := service.TakeOffEquipmentItem(ctx, user.ID, "invalid_slot")
		assert.ErrorIs(t, err, ErrInvalidEquipmentType)
	})

	t.Run("unequip one item with multiple equipped", func(t *testing.T) {
		ts := time.Now().UnixNano()
		locationID := uuid.New()
		locationQuery := `INSERT INTO locations (id, name, slug, cell, inactive) VALUES ($1, $2, $3, $4, $5)`
		_, err := db.Exec(locationQuery, locationID, "Test Location 2", fmt.Sprintf("test_location_2_%d", ts), false, false)
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
		_, err = db.Exec(itemQuery, weaponID, "Test Weapon", fmt.Sprintf("test-weapon-%d", ts), 10, 0, 0, 1, 100, weaponCatID)
		require.NoError(t, err)

		chestID := uuid.New()
		_, err = db.Exec(itemQuery, chestID, "Test Chest", fmt.Sprintf("test-chest-%d", ts), 0, 15, 30, 1, 100, chestCatID)
		require.NoError(t, err)

		ts = time.Now().UnixNano()
		multiUser := &domain.User{
			Username:   fmt.Sprintf("m%d", ts%1000000),
			Email:      fmt.Sprintf("multi%d@example.com", ts),
			Password:   "password",
			LocationID: locationID,
			Attack:     11,
			Defense:    16,
			Hp:         50,
			CurrentHp:  50,
			Level:      5,
		}
		err = userRepo.Create(multiUser)
		require.NoError(t, err)
		_, err = db.Exec(`UPDATE users SET weapon_equipment_item_id = $1, chest_equipment_item_id = $2 WHERE id = $3`, weaponID, chestID, multiUser.ID)
		require.NoError(t, err)

		err = service.TakeOffEquipmentItem(ctx, multiUser.ID, "weapon")
		require.NoError(t, err)

		type stats struct {
			Attack  uint `db:"attack"`
			Defense uint `db:"defense"`
			Hp      uint `db:"hp"`
		}
		var userStats stats
		statsQuery := `SELECT attack, defense, hp FROM users WHERE id = $1`
		err = db.Get(&userStats, statsQuery, multiUser.ID)
		require.NoError(t, err)

		assert.Equal(t, uint(1), userStats.Attack)
		assert.Equal(t, uint(16), userStats.Defense)
		assert.Equal(t, uint(50), userStats.Hp)

		var chestEquippedID uuid.UUID
		chestQuery := `SELECT chest_equipment_item_id FROM users WHERE id = $1`
		err = db.Get(&chestEquippedID, chestQuery, multiUser.ID)
		require.NoError(t, err)
		assert.Equal(t, chestID, chestEquippedID)
	})

	t.Run("current hp stays unchanged when below new max hp", func(t *testing.T) {
		locationID := uuid.New()
		_, err := db.Exec(`INSERT INTO locations (id, name, slug, cell, inactive) VALUES ($1, $2, $3, $4, $5)`, locationID, "Test Location 3", fmt.Sprintf("test_location_3_%d", time.Now().UnixNano()), false, false)
		require.NoError(t, err)

		weaponCatID := uuid.New()
		_, err = db.Exec(`INSERT INTO equipment_categories (id, name, type) VALUES ($1, $2, $3::equipment_category_type)`, weaponCatID, "Weapon", "weapon")
		require.NoError(t, err)

		weaponID := uuid.New()
		_, err = db.Exec(`INSERT INTO equipment_items (id, name, slug, attack, defense, hp, required_level, price, equipment_category_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			weaponID, "Test Weapon 2", fmt.Sprintf("test-weapon-2-%d", time.Now().UnixNano()), 0, 0, 20, 1, 100, weaponCatID)
		require.NoError(t, err)

		ts := time.Now().UnixNano()
		testUser := &domain.User{
			Username:   fmt.Sprintf("h%d", ts%1000000),
			Email:      fmt.Sprintf("hpuser%d@example.com", ts),
			Password:   "password",
			LocationID: locationID,
			Attack:     1,
			Defense:    1,
			Hp:         40,
			CurrentHp:  15,
			Level:      5,
		}
		err = userRepo.Create(testUser)
		require.NoError(t, err)
		_, err = db.Exec(`UPDATE users SET weapon_equipment_item_id = $1 WHERE id = $2`, weaponID, testUser.ID)
		require.NoError(t, err)

		err = service.TakeOffEquipmentItem(ctx, testUser.ID, "weapon")
		require.NoError(t, err)

		var currentHp int
		var hp int
		err = db.Get(&currentHp, `SELECT current_hp FROM users WHERE id = $1`, testUser.ID)
		require.NoError(t, err)
		err = db.Get(&hp, `SELECT hp FROM users WHERE id = $1`, testUser.ID)
		require.NoError(t, err)

		assert.Equal(t, 15, currentHp)
		assert.Equal(t, 20, hp)
	})
}
