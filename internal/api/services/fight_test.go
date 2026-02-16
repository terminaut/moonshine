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

func setupFightTestData(db *sqlx.DB) (*domain.Location, *domain.User, *domain.Bot, *domain.Fight, error) {
	location := &domain.Location{
		Name: fmt.Sprintf("Test Location %d", time.Now().UnixNano()),
		Slug: fmt.Sprintf("test-location-%d", time.Now().UnixNano()),
		Cell: false,
	}
	locationRepo := repository.NewLocationRepository(db)
	if err := locationRepo.Create(location); err != nil {
		return nil, nil, nil, nil, err
	}

	user := &domain.User{
		Username:   fmt.Sprintf("testuser%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("test%d@example.com", time.Now().UnixNano()),
		Password:   "password",
		LocationID: location.ID,
		Attack:     10,
		Defense:    5,
		Hp:         100,
		CurrentHp:  100,
		Level:      1,
	}
	userRepo := repository.NewUserRepository(db)
	if err := userRepo.Create(user); err != nil {
		return nil, nil, nil, nil, err
	}

	bot := &domain.Bot{
		Name:    "Test Bot",
		Slug:    fmt.Sprintf("test-bot-%d", time.Now().UnixNano()),
		Attack:  8,
		Defense: 4,
		Hp:      80,
		Level:   1,
		Avatar:  "images/bots/test",
	}
	botRepo := repository.NewBotRepository(db)
	if err := botRepo.Create(bot); err != nil {
		return nil, nil, nil, nil, err
	}

	linkID := uuid.New()
	linkQuery := `INSERT INTO location_bots (id, location_id, bot_id) VALUES ($1, $2, $3)`
	_, err := db.Exec(linkQuery, linkID, location.ID, bot.ID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	fight := &domain.Fight{
		UserID: user.ID,
		BotID:  bot.ID,
		Status: domain.FightStatusInProgress,
	}
	fightRepo := repository.NewFightRepository(db)
	fightID, err := fightRepo.Create(fight)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	fight.ID = fightID

	roundRepo := repository.NewRoundRepository(db)
	err = roundRepo.Create(fightID, user.CurrentHp, bot.Hp)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return location, user, bot, fight, nil
}

func TestFightService_GetCurrentFight(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	db := testDB.DB()
	service := NewFightService(
		db,
		repository.NewFightRepository(db),
		repository.NewBotRepository(db),
		repository.NewUserRepository(db),
		repository.NewRoundRepository(db),
	)
	ctx := context.Background()

	t.Run("successfully get current fight", func(t *testing.T) {
		_, user, bot, fight, err := setupFightTestData(db)
		require.NoError(t, err)

		result, err := service.GetCurrentFight(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, user.ID, result.User.ID)
		assert.Equal(t, user.Username, result.User.Username)
		assert.Equal(t, user.Hp, result.User.Hp)
		assert.Equal(t, user.CurrentHp, result.User.CurrentHp)

		assert.Equal(t, bot.ID, result.Bot.ID)
		assert.Equal(t, bot.Name, result.Bot.Name)
		assert.Equal(t, bot.Hp, result.Bot.Hp)

		assert.Equal(t, fight.ID, result.Fight.ID)
		assert.Equal(t, fight.UserID, result.Fight.UserID)
		assert.Equal(t, fight.BotID, result.Fight.BotID)
		assert.Equal(t, domain.FightStatusInProgress, result.Fight.Status)

		require.NotEmpty(t, result.Fight.Rounds)
		assert.Equal(t, user.CurrentHp, result.Fight.Rounds[0].PlayerHp)
		assert.Equal(t, int(bot.Hp), result.Fight.Rounds[0].BotHp)
		assert.Equal(t, domain.RoundStatusInProgress, result.Fight.Rounds[0].Status)
	})

	t.Run("non-existent user returns error", func(t *testing.T) {
		_, err := service.GetCurrentFight(ctx, uuid.New())
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
	})

	t.Run("user without active fight returns error", func(t *testing.T) {
		locationID := uuid.New()
		location := &domain.Location{
			Model: domain.Model{ID: locationID},
			Name:  fmt.Sprintf("Test Location %d", time.Now().UnixNano()),
			Slug:  fmt.Sprintf("test-location-%d", time.Now().UnixNano()),
			Cell:  false,
		}
		locationRepo := repository.NewLocationRepository(db)
		require.NoError(t, locationRepo.Create(location))

		user := &domain.User{
			Username:   fmt.Sprintf("testuser%d", time.Now().UnixNano()),
			Email:      fmt.Sprintf("test%d@example.com", time.Now().UnixNano()),
			Password:   "password",
			LocationID: location.ID,
			Attack:     10,
			Defense:    5,
			Hp:         100,
			CurrentHp:  100,
			Level:      1,
		}
		userRepo := repository.NewUserRepository(db)
		require.NoError(t, userRepo.Create(user))

		_, err := service.GetCurrentFight(ctx, user.ID)
		assert.Error(t, err)
		assert.Equal(t, ErrNoActiveFight, err)
	})
}

func TestFightService_Hit(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	db := testDB.DB()
	service := NewFightService(
		db,
		repository.NewFightRepository(db),
		repository.NewBotRepository(db),
		repository.NewUserRepository(db),
		repository.NewRoundRepository(db),
	)
	ctx := context.Background()

	t.Run("successfully hit when both have HP remaining", func(t *testing.T) {
		_, user, bot, fight, err := setupFightTestData(db)
		require.NoError(t, err)

		initialUserHp := user.CurrentHp
		initialBotHp := int(bot.Hp)

		result, err := service.Hit(ctx, user.ID, "HEAD", "CHEST")
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, user.ID, result.User.ID)
		assert.Equal(t, bot.ID, result.Bot.ID)
		assert.Equal(t, fight.ID, result.Fight.ID)

		require.NotEmpty(t, result.Fight.Rounds)

		finishedRounds := make([]*domain.Round, 0)
		for _, round := range result.Fight.Rounds {
			if round.Status == domain.RoundStatusFinished {
				finishedRounds = append(finishedRounds, round)
			}
		}

		require.NotEmpty(t, finishedRounds, "should have at least one finished round")
		lastFinishedRound := finishedRounds[len(finishedRounds)-1]

		assert.GreaterOrEqual(t, lastFinishedRound.PlayerDamage, uint(0))
		assert.GreaterOrEqual(t, lastFinishedRound.BotDamage, uint(0))
		assert.NotNil(t, lastFinishedRound.BotAttackPoint)
		assert.NotNil(t, lastFinishedRound.BotDefensePoint)

		inProgressRounds := make([]*domain.Round, 0)
		for _, round := range result.Fight.Rounds {
			if round.Status == domain.RoundStatusInProgress {
				inProgressRounds = append(inProgressRounds, round)
			}
		}

		require.NotEmpty(t, inProgressRounds, "should have at least one in-progress round")
		lastRound := inProgressRounds[len(inProgressRounds)-1]

		assert.LessOrEqual(t, lastRound.PlayerHp, initialUserHp, "player HP should decrease or stay same")
		assert.LessOrEqual(t, lastRound.BotHp, initialBotHp, "bot HP should decrease or stay same")
		assert.Greater(t, lastRound.PlayerHp, 0, "player should still have HP")
		assert.Greater(t, lastRound.BotHp, 0, "bot should still have HP")
	})

	t.Run("invalid attack point returns error", func(t *testing.T) {
		_, user, _, _, err := setupFightTestData(db)
		require.NoError(t, err)

		_, err = service.Hit(ctx, user.ID, "INVALID", "HEAD")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidBodyPart, err)
	})

	t.Run("invalid defense point returns error", func(t *testing.T) {
		_, user, _, _, err := setupFightTestData(db)
		require.NoError(t, err)

		_, err = service.Hit(ctx, user.ID, "HEAD", "INVALID")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidBodyPart, err)
	})

	t.Run("non-existent user returns error", func(t *testing.T) {
		_, err := service.Hit(ctx, uuid.New(), "HEAD", "CHEST")
		assert.Error(t, err)
		assert.Equal(t, ErrNoActiveFight, err)
	})

	t.Run("user without active fight returns error", func(t *testing.T) {
		locationID := uuid.New()
		location := &domain.Location{
			Model: domain.Model{ID: locationID},
			Name:  fmt.Sprintf("Test Location %d", time.Now().UnixNano()),
			Slug:  fmt.Sprintf("test-location-%d", time.Now().UnixNano()),
			Cell:  false,
		}
		locationRepo := repository.NewLocationRepository(db)
		require.NoError(t, locationRepo.Create(location))

		user := &domain.User{
			Username:   fmt.Sprintf("testuser%d", time.Now().UnixNano()),
			Email:      fmt.Sprintf("test%d@example.com", time.Now().UnixNano()),
			Password:   "password",
			LocationID: location.ID,
			Attack:     10,
			Defense:    5,
			Hp:         100,
			CurrentHp:  100,
			Level:      1,
		}
		userRepo := repository.NewUserRepository(db)
		require.NoError(t, userRepo.Create(user))

		_, err := service.Hit(ctx, user.ID, "HEAD", "CHEST")
		assert.Error(t, err)
		assert.Equal(t, ErrNoActiveFight, err)
	})
}
