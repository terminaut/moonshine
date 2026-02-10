package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"context"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

func setupFightHandlerTest(t *testing.T) (*FightHandler, *sqlx.DB, *domain.User) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}
	db := testDB.DB()
	handler := NewFightHandler(db)

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

	return handler, db, user
}

func setupFightWithBot(t *testing.T, db *sqlx.DB, user *domain.User) (*domain.Bot, *domain.Fight, error) {
	botID := uuid.New()
	bot := &domain.Bot{
		Model:   domain.Model{ID: botID},
		Name:    "Test Bot",
		Slug:    fmt.Sprintf("test-bot-%d", time.Now().UnixNano()),
		Attack:  8,
		Defense: 4,
		Hp:      80,
		Level:   1,
		Avatar:  "images/bots/test",
	}
	botRepo := repository.NewBotRepository(db)
	require.NoError(t, botRepo.Create(bot))

	linkID := uuid.New()
	linkQuery := `INSERT INTO location_bots (id, location_id, bot_id) VALUES ($1, $2, $3)`
	_, err := db.Exec(linkQuery, linkID, user.LocationID, botID)
	require.NoError(t, err)

	fight := &domain.Fight{
		UserID: user.ID,
		BotID:  bot.ID,
		Status: domain.FightStatusInProgress,
	}
	fightRepo := repository.NewFightRepository(db)
	fightID, err := fightRepo.Create(fight)
	require.NoError(t, err)
	fight.ID = fightID

	roundRepo := repository.NewRoundRepository(db)
	err = roundRepo.Create(fightID, user.CurrentHp, bot.Hp)
	require.NoError(t, err)

	return bot, fight, nil
}

func TestFightHandler_GetCurrentFight(t *testing.T) {
	handler, db, user := setupFightHandlerTest(t)

	t.Run("successfully get current fight", func(t *testing.T) {
		bot, fight, err := setupFightWithBot(t, db, user)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/fights/current", nil)
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, user.ID)
		req = req.WithContext(ctx)
		c := e.NewContext(req, rec)

		err = handler.GetCurrentFight(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response GetCurrentFightResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, user.ID.String(), response.User.ID)
		assert.Equal(t, user.Username, response.User.Username)
		assert.Equal(t, int(user.Hp), response.User.Hp)
		assert.Equal(t, int(user.CurrentHp), response.User.CurrentHp)
		assert.True(t, response.User.InFight)

		assert.Equal(t, bot.ID.String(), response.Bot.ID)
		assert.Equal(t, bot.Name, response.Bot.Name)
		assert.Equal(t, int(bot.Hp), response.Bot.Hp)

		assert.Equal(t, fight.ID.String(), response.Fight.ID)
		assert.Equal(t, user.ID.String(), response.Fight.UserID)
		assert.Equal(t, bot.ID.String(), response.Fight.BotID)
		assert.Equal(t, string(domain.FightStatusInProgress), response.Fight.Status)

		require.NotEmpty(t, response.Fight.Rounds)
		assert.Equal(t, int(user.CurrentHp), response.Fight.Rounds[0].PlayerHp)
		assert.Equal(t, int(bot.Hp), response.Fight.Rounds[0].BotHp)
	})

	t.Run("user without active fight returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/fights/current", nil)
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, user.ID)
		req = req.WithContext(ctx)
		c := e.NewContext(req, rec)

		err := handler.GetCurrentFight(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		var response map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "no active fight")
	})
}

func TestFightHandler_Hit(t *testing.T) {
	handler, db, user := setupFightHandlerTest(t)

	t.Run("successfully hit when both have HP remaining", func(t *testing.T) {
		bot, fight, err := setupFightWithBot(t, db, user)
		require.NoError(t, err)

		roundRepo := repository.NewRoundRepository(db)
		rounds, err := roundRepo.FindByFightID(fight.ID)
		require.NoError(t, err)
		require.NotEmpty(t, rounds)
		initialPlayerHp := rounds[0].PlayerHp
		initialBotHp := rounds[0].BotHp

		hitRequest := HitRequest{
			Attack:  "HEAD",
			Defense: "CHEST",
		}
		body, err := json.Marshal(hitRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/fights/current/hit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, user.ID)
		req = req.WithContext(ctx)
		c := e.NewContext(req, rec)

		err = handler.Hit(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response GetCurrentFightResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, user.ID.String(), response.User.ID)
		assert.Equal(t, bot.ID.String(), response.Bot.ID)
		assert.Equal(t, fight.ID.String(), response.Fight.ID)

		require.NotEmpty(t, response.Fight.Rounds)

		finishedRounds := make([]*dto.Round, 0)
		for _, round := range response.Fight.Rounds {
			if round.Status == "FINISHED" {
				finishedRounds = append(finishedRounds, round)
			}
		}

		require.NotEmpty(t, finishedRounds, "should have at least one finished round")
		lastFinishedRound := finishedRounds[len(finishedRounds)-1]

		assert.GreaterOrEqual(t, lastFinishedRound.PlayerDamage, 0)
		assert.GreaterOrEqual(t, lastFinishedRound.BotDamage, 0)
		assert.NotNil(t, lastFinishedRound.BotAttackPoint)
		assert.NotNil(t, lastFinishedRound.BotDefensePoint)

		inProgressRounds := make([]*dto.Round, 0)
		for _, round := range response.Fight.Rounds {
			if round.Status == "IN_PROGRESS" {
				inProgressRounds = append(inProgressRounds, round)
			}
		}

		require.NotEmpty(t, inProgressRounds, "should have at least one in-progress round")
		lastRound := inProgressRounds[len(inProgressRounds)-1]

		assert.LessOrEqual(t, lastRound.PlayerHp, int(initialPlayerHp), "player HP should decrease or stay same")
		assert.LessOrEqual(t, lastRound.BotHp, int(initialBotHp), "bot HP should decrease or stay same")
		assert.Greater(t, lastRound.PlayerHp, 0, "player should still have HP")
		assert.Greater(t, lastRound.BotHp, 0, "bot should still have HP")
	})

	t.Run("invalid attack point returns 400", func(t *testing.T) {
		_, _, err := setupFightWithBot(t, db, user)
		require.NoError(t, err)

		hitRequest := HitRequest{
			Attack:  "INVALID",
			Defense: "HEAD",
		}
		body, err := json.Marshal(hitRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/fights/current/hit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, user.ID)
		req = req.WithContext(ctx)
		c := e.NewContext(req, rec)

		err = handler.Hit(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "invalid body part")
	})

	t.Run("invalid defense point returns 400", func(t *testing.T) {
		_, _, err := setupFightWithBot(t, db, user)
		require.NoError(t, err)

		hitRequest := HitRequest{
			Attack:  "HEAD",
			Defense: "INVALID",
		}
		body, err := json.Marshal(hitRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/fights/current/hit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, user.ID)
		req = req.WithContext(ctx)
		c := e.NewContext(req, rec)

		err = handler.Hit(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "invalid body part")
	})

	t.Run("missing attack field returns 400", func(t *testing.T) {
		hitRequest := map[string]string{
			"defense": "HEAD",
		}
		body, err := json.Marshal(hitRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/fights/current/hit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, user.ID)
		req = req.WithContext(ctx)
		c := e.NewContext(req, rec)

		err = handler.Hit(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("user without active fight returns 404", func(t *testing.T) {
		hitRequest := HitRequest{
			Attack:  "HEAD",
			Defense: "CHEST",
		}
		body, err := json.Marshal(hitRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/fights/current/hit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, user.ID)
		req = req.WithContext(ctx)
		c := e.NewContext(req, rec)

		err = handler.Hit(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		var response map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "no active fight")
	})
}
