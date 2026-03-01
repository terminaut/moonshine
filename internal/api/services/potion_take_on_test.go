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

func setupPotionTakeOnTestData(t *testing.T) (*domain.User, *domain.Potion) {
	t.Helper()
	testutil.RequireDB(t, testDB)
	ts := time.Now().UnixNano()

	locationRepo := repository.NewLocationRepository(testDB)
	location := &domain.Location{
		Name: fmt.Sprintf("PotionTakeOnLoc %d", ts),
		Slug: fmt.Sprintf("potion-takeon-loc-%d", ts),
	}
	require.NoError(t, locationRepo.Create(location))

	userRepo := repository.NewUserRepository(testDB)
	user := &domain.User{
		Username:   fmt.Sprintf("potion-takeon%d", ts%1000000),
		Email:      fmt.Sprintf("potion-takeon%d@test.com", ts),
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
		Name:  fmt.Sprintf("TakeOn Potion %d", ts),
		Slug:  fmt.Sprintf("takeon-potion-%d", ts),
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

func addPotionToInventory(t *testing.T, userID, potionID uuid.UUID) {
	t.Helper()
	_, err := testDB.Exec(
		`INSERT INTO inventory (id, user_id, potion_id) VALUES ($1, $2, $3)`,
		uuid.New(), userID, potionID,
	)
	require.NoError(t, err)
}

func TestPotionTakeOnService_TakeOnPotion(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	newService := func() *PotionTakeOnService {
		return NewPotionTakeOnService(
			testDB,
			repository.NewPotionRepository(testDB),
			repository.NewInventoryRepository(testDB),
			repository.NewUserRepository(testDB),
		)
	}

	t.Run("equip to empty slot 1", func(t *testing.T) {
		user, potion := setupPotionTakeOnTestData(t)
		addPotionToInventory(t, user.ID, potion.ID)

		err := newService().TakeOnPotion(ctx, user.ID, potion.ID)
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		require.NotNil(t, userAfter.Potion1ID)
		assert.Equal(t, potion.ID, *userAfter.Potion1ID)
		assert.Nil(t, userAfter.Potion2ID)
		assert.Nil(t, userAfter.Potion3ID)

		var count int
		err = testDB.Get(&count, `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND potion_id = $2 AND deleted_at IS NULL`, user.ID, potion.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("equip fills slots sequentially", func(t *testing.T) {
		user, potion := setupPotionTakeOnTestData(t)

		ts := time.Now().UnixNano()
		potion2 := &domain.Potion{
			Name:  fmt.Sprintf("Potion2 %d", ts),
			Slug:  fmt.Sprintf("potion2-%d", ts),
			Stat:  "ATTACK",
			Value: 10,
			Price: 50,
		}
		potion2.ID = uuid.New()
		err := testDB.QueryRow(
			`INSERT INTO potions (id, name, slug, stat, value, price) VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at`,
			potion2.ID, potion2.Name, potion2.Slug, potion2.Stat, potion2.Value, potion2.Price,
		).Scan(&potion2.CreatedAt)
		require.NoError(t, err)

		addPotionToInventory(t, user.ID, potion.ID)
		addPotionToInventory(t, user.ID, potion2.ID)

		err = newService().TakeOnPotion(ctx, user.ID, potion.ID)
		require.NoError(t, err)

		err = newService().TakeOnPotion(ctx, user.ID, potion2.ID)
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		require.NotNil(t, userAfter.Potion1ID)
		require.NotNil(t, userAfter.Potion2ID)
		assert.Equal(t, potion.ID, *userAfter.Potion1ID)
		assert.Equal(t, potion2.ID, *userAfter.Potion2ID)
		assert.Nil(t, userAfter.Potion3ID)
	})

	t.Run("all slots full replaces slot 1", func(t *testing.T) {
		user, potion := setupPotionTakeOnTestData(t)

		potions := make([]*domain.Potion, 3)
		for i := range 3 {
			ts := time.Now().UnixNano() + int64(i)
			p := &domain.Potion{
				Name:  fmt.Sprintf("Fill Potion %d", ts),
				Slug:  fmt.Sprintf("fill-potion-%d", ts),
				Stat:  "CURRENT_HP",
				Value: 10,
				Price: 10,
			}
			p.ID = uuid.New()
			err := testDB.QueryRow(
				`INSERT INTO potions (id, name, slug, stat, value, price) VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at`,
				p.ID, p.Name, p.Slug, p.Stat, p.Value, p.Price,
			).Scan(&p.CreatedAt)
			require.NoError(t, err)
			potions[i] = p
			addPotionToInventory(t, user.ID, p.ID)
		}

		svc := newService()
		for i := range 3 {
			err := svc.TakeOnPotion(ctx, user.ID, potions[i].ID)
			require.NoError(t, err)
		}

		addPotionToInventory(t, user.ID, potion.ID)
		err := svc.TakeOnPotion(ctx, user.ID, potion.ID)
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		require.NotNil(t, userAfter.Potion1ID)
		assert.Equal(t, potion.ID, *userAfter.Potion1ID)

		var count int
		err = testDB.Get(&count, `SELECT COUNT(*) FROM inventory WHERE user_id = $1 AND potion_id = $2 AND deleted_at IS NULL`, user.ID, potions[0].ID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("not in inventory", func(t *testing.T) {
		user, potion := setupPotionTakeOnTestData(t)

		err := newService().TakeOnPotion(ctx, user.ID, potion.ID)
		assert.ErrorIs(t, err, ErrItemNotInInventory)
	})

	t.Run("user not found", func(t *testing.T) {
		_, potion := setupPotionTakeOnTestData(t)

		err := newService().TakeOnPotion(ctx, uuid.New(), potion.ID)
		assert.ErrorIs(t, err, repository.ErrUserNotFound)
	})
}
