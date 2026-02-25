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

func setupPotionUseTestData(t *testing.T) (*domain.User, *domain.Potion, *domain.Fight) {
	t.Helper()
	testutil.RequireDB(t, testDB)
	ts := time.Now().UnixNano()

	locationRepo := repository.NewLocationRepository(testDB)
	location := &domain.Location{
		Name: fmt.Sprintf("PotionUseLoc %d", ts),
		Slug: fmt.Sprintf("potion-use-loc-%d", ts),
	}
	require.NoError(t, locationRepo.Create(location))

	userRepo := repository.NewUserRepository(testDB)
	user := &domain.User{
		Username:   fmt.Sprintf("potion-user%d", ts%1000000),
		Email:      fmt.Sprintf("potion-user%d@test.com", ts),
		Password:   "pass",
		LocationID: location.ID,
		Gold:       500,
		Hp:         200,
		CurrentHp:  100,
		Level:      1,
		Attack:     10,
		Defense:    5,
	}
	require.NoError(t, userRepo.Create(user))

	potion := &domain.Potion{
		Name:  fmt.Sprintf("Use Potion %d", ts),
		Slug:  fmt.Sprintf("use-potion-%d", ts),
		Stat:  "CURRENT_HP",
		Value: 50,
		Price: 100,
	}
	potion.ID = uuid.New()
	err := testDB.QueryRow(
		`INSERT INTO potions (id, name, slug, stat, value, price) VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at`,
		potion.ID, potion.Name, potion.Slug, potion.Stat, potion.Value, potion.Price,
	).Scan(&potion.CreatedAt)
	require.NoError(t, err)

	bot := &domain.Bot{
		Name:    fmt.Sprintf("Use Bot %d", ts),
		Slug:    fmt.Sprintf("use-bot-%d", ts),
		Attack:  8,
		Defense: 4,
		Hp:      80,
		Level:   1,
		Avatar:  "images/bots/test",
	}
	botRepo := repository.NewBotRepository(testDB)
	require.NoError(t, botRepo.Create(bot))

	linkID := uuid.New()
	_, err = testDB.Exec(`INSERT INTO location_bots (id, location_id, bot_id) VALUES ($1, $2, $3)`, linkID, location.ID, bot.ID)
	require.NoError(t, err)

	fight := &domain.Fight{
		UserID: user.ID,
		BotID:  bot.ID,
		Status: domain.FightStatusInProgress,
	}
	fightRepo := repository.NewFightRepository(testDB)
	fightID, err := fightRepo.Create(fight)
	require.NoError(t, err)
	fight.ID = fightID

	roundRepo := repository.NewRoundRepository(testDB)
	err = roundRepo.Create(fightID, user.CurrentHp, bot.Hp)
	require.NoError(t, err)

	return user, potion, fight
}

func TestPotionUseService_UsePotion(t *testing.T) {
	testutil.RequireDB(t, testDB)
	ctx := context.Background()

	newService := func() *PotionUseService {
		return NewPotionUseService(
			testDB,
			repository.NewPotionRepository(testDB),
			repository.NewUserRepository(testDB),
		)
	}

	t.Run("heals player in fight", func(t *testing.T) {
		user, potion, fight := setupPotionUseTestData(t)
		equipPotionToSlot(t, user.ID, potion.ID, "potion1")

		err := newService().UsePotion(ctx, user.ID, "potion1")
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		assert.Nil(t, userAfter.Potion1ID)
		assert.Equal(t, 150, userAfter.CurrentHp)

		roundRepo := repository.NewRoundRepository(testDB)
		rounds, err := roundRepo.FindByFightID(fight.ID)
		require.NoError(t, err)
		require.NotEmpty(t, rounds)
		assert.Equal(t, 150, rounds[0].PlayerHp)
	})

	t.Run("does not exceed max hp", func(t *testing.T) {
		user, potion, fight := setupPotionUseTestData(t)

		_, err := testDB.Exec(`UPDATE users SET current_hp = hp WHERE id = $1`, user.ID)
		require.NoError(t, err)

		roundRepo := repository.NewRoundRepository(testDB)
		rounds, err := roundRepo.FindByFightID(fight.ID)
		require.NoError(t, err)
		_, err = testDB.Exec(`UPDATE rounds SET player_hp = $1 WHERE id = $2`, 200, rounds[0].ID)
		require.NoError(t, err)

		equipPotionToSlot(t, user.ID, potion.ID, "potion1")

		err = newService().UsePotion(ctx, user.ID, "potion1")
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, int(user.Hp), userAfter.CurrentHp)
	})

	t.Run("clears potion slot after use", func(t *testing.T) {
		user, potion, _ := setupPotionUseTestData(t)
		equipPotionToSlot(t, user.ID, potion.ID, "potion2")

		err := newService().UsePotion(ctx, user.ID, "potion2")
		require.NoError(t, err)

		userAfter, err := repository.NewUserRepository(testDB).FindByID(user.ID)
		require.NoError(t, err)
		assert.Nil(t, userAfter.Potion2ID)
	})

	t.Run("no potion equipped", func(t *testing.T) {
		user, _, _ := setupPotionUseTestData(t)

		err := newService().UsePotion(ctx, user.ID, "potion1")
		assert.ErrorIs(t, err, ErrNoItemEquipped)
	})

	t.Run("invalid slot", func(t *testing.T) {
		user, _, _ := setupPotionUseTestData(t)

		err := newService().UsePotion(ctx, user.ID, "invalid")
		assert.ErrorIs(t, err, ErrInvalidEquipmentType)
	})

	t.Run("user not found", func(t *testing.T) {
		err := newService().UsePotion(ctx, uuid.New(), "potion1")
		assert.ErrorIs(t, err, repository.ErrUserNotFound)
	})
}
