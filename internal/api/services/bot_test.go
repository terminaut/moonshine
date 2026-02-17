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

func setupBotTestData(db *sqlx.DB) (*domain.Location, *domain.Bot, error) {
	location := &domain.Location{
		Name:  fmt.Sprintf("Test Location %d", time.Now().UnixNano()),
		Slug:  fmt.Sprintf("test-location-%d", time.Now().UnixNano()),
		Cell:  false,
	}
	locationRepo := repository.NewLocationRepository(db)
	if err := locationRepo.Create(location); err != nil {
		return nil, nil, err
	}

	bot := &domain.Bot{
		Name:    "Test Bot",
		Slug:    fmt.Sprintf("test-bot-%d", time.Now().UnixNano()),
		Attack:  5,
		Defense: 3,
		Hp:      20,
		Level:   1,
		Avatar:  "images/bots/test",
	}
	botRepo := repository.NewBotRepository(db)
	if err := botRepo.Create(bot); err != nil {
		return nil, nil, err
	}

	linkID := uuid.New()
	linkQuery := `INSERT INTO location_bots (id, location_id, bot_id) VALUES ($1, $2, $3)`
	_, err := db.Exec(linkQuery, linkID, location.ID, bot.ID)
	if err != nil {
		return nil, nil, err
	}

	return location, bot, nil
}

func TestBotService_GetBotsByLocationSlug(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	db := testDB
	service := NewBotService(
		repository.NewLocationRepository(db),
		repository.NewBotRepository(db),
		repository.NewUserRepository(db),
		repository.NewFightRepository(db),
		repository.NewRoundRepository(db),
	)

	t.Run("successfully get bots by location slug", func(t *testing.T) {
		location, bot, err := setupBotTestData(db)
		require.NoError(t, err)

		bots, err := service.GetBotsByLocationSlug(location.Slug)
		require.NoError(t, err)
		assert.NotEmpty(t, bots)
		assert.Equal(t, bot.Slug, bots[0].Slug)
		assert.Equal(t, bot.Name, bots[0].Name)
	})

	t.Run("empty location slug returns error", func(t *testing.T) {
		_, err := service.GetBotsByLocationSlug("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "location slug is required")
	})

	t.Run("non-existent location returns error", func(t *testing.T) {
		_, err := service.GetBotsByLocationSlug("non-existent-location")
		assert.Error(t, err)
		assert.ErrorIs(t, err, repository.ErrLocationNotFound)
	})

	t.Run("location with no bots returns empty list", func(t *testing.T) {
		locationID := uuid.New()
		location := &domain.Location{
			Model: domain.Model{ID: locationID},
			Name:  fmt.Sprintf("Empty Location %d", time.Now().UnixNano()),
			Slug:  fmt.Sprintf("empty-location-%d", time.Now().UnixNano()),
			Cell:  false,
		}
		locationRepo := repository.NewLocationRepository(db)
		require.NoError(t, locationRepo.Create(location))

		bots, err := service.GetBotsByLocationSlug(location.Slug)
		require.NoError(t, err)
		assert.Empty(t, bots)
	})
}

func TestBotService_Attack(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	db := testDB
	service := NewBotService(
		repository.NewLocationRepository(db),
		repository.NewBotRepository(db),
		repository.NewUserRepository(db),
		repository.NewFightRepository(db),
		repository.NewRoundRepository(db),
	)
	ctx := context.Background()

	setupAttackTestData := func(db *sqlx.DB) (*domain.Location, *domain.User, *domain.Bot, error) {
		location, bot, err := setupBotTestData(db)
		if err != nil {
			return nil, nil, nil, err
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
			Level:      1,
		}
		userRepo := repository.NewUserRepository(db)
		if err := userRepo.Create(user); err != nil {
			return nil, nil, nil, err
		}

		return location, user, bot, nil
	}

	t.Run("successfully attack bot", func(t *testing.T) {
		_, user, bot, err := setupAttackTestData(db)
		require.NoError(t, err)

		result, err := service.Attack(ctx, bot.Slug, user.ID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, user.ID, result.User.ID)
		assert.Equal(t, bot.ID, result.Bot.ID)

		fightRepo := repository.NewFightRepository(db)
		fight, err := fightRepo.FindActiveByUserID(user.ID)
		require.NoError(t, err)
		require.NotNil(t, fight)
		assert.Equal(t, user.ID, fight.UserID)
		assert.Equal(t, bot.ID, fight.BotID)
		assert.Equal(t, domain.FightStatusInProgress, fight.Status)

		var roundCount int
		roundQuery := `SELECT COUNT(*) FROM rounds WHERE fight_id = $1 AND deleted_at IS NULL`
		err = db.Get(&roundCount, roundQuery, fight.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, roundCount, "should create exactly one round")
	})

	t.Run("empty bot slug returns error", func(t *testing.T) {
		_, user, _, err := setupAttackTestData(db)
		require.NoError(t, err)

		_, err = service.Attack(ctx, "", user.ID)
		assert.Error(t, err)
	})

	t.Run("non-existent bot returns error", func(t *testing.T) {
		_, user, _, err := setupAttackTestData(db)
		require.NoError(t, err)

		_, err = service.Attack(ctx, "non-existent-bot", user.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, repository.ErrBotNotFound)
	})

	t.Run("non-existent user returns error", func(t *testing.T) {
		_, _, bot, err := setupAttackTestData(db)
		require.NoError(t, err)

		_, err = service.Attack(ctx, bot.Slug, uuid.New())
		assert.Error(t, err)
		assert.ErrorIs(t, err, repository.ErrUserNotFound)
	})
}
