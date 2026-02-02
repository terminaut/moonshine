package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"moonshine/internal/config"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

var testDB *repository.Database

func TestMain(m *testing.M) {
	err := godotenv.Load("../../../.env.test")
	if err != nil {
	}
	cfg := config.Load()

	db, err := repository.New(cfg)
	if err != nil {
		testDB = nil
		code := m.Run()
		os.Exit(code)
	}
	testDB = db

	code := m.Run()

	if testDB != nil {
		testDB.Close()
	}
	os.Exit(code)
}

func setupBotHandlerTest(t *testing.T) (*BotHandler, *sqlx.DB, echo.Context) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}
	db := testDB.DB()
	handler := NewBotHandler(db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	return handler, db, c
}

func TestBotHandler_GetBots(t *testing.T) {
	handler, db, _ := setupBotHandlerTest(t)

	t.Run("empty location slug returns bad request", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.SetPath("/api/locations/:location_slug/bots")
		c.SetParamNames("location_slug")
		c.SetParamValues("")

		err := handler.GetBots(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "location slug is required")
	})

	t.Run("non-existent location returns internal server error", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.SetPath("/api/locations/:location_slug/bots")
		c.SetParamNames("location_slug")
		c.SetParamValues("non-existent")

		err := handler.GetBots(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("successfully get bots by location slug", func(t *testing.T) {
		location := &domain.Location{
			Name:     fmt.Sprintf("Test Location %d", time.Now().UnixNano()),
			Slug:     fmt.Sprintf("test-location-%d", time.Now().UnixNano()),
			Cell:     false,
			Inactive: false,
		}
		locationRepo := repository.NewLocationRepository(db)
		err := locationRepo.Create(location)
		require.NoError(t, err)

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
		err = botRepo.Create(bot)
		require.NoError(t, err)

		linkID := uuid.New()
		linkQuery := `INSERT INTO location_bots (id, location_id, bot_id) VALUES ($1, $2, $3)`
		_, err = db.Exec(linkQuery, linkID, location.ID, bot.ID)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.SetPath("/api/locations/:location_slug/bots")
		c.SetParamNames("location_slug")
		c.SetParamValues(location.Slug)

		err = handler.GetBots(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response BotResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Bots)
		assert.Equal(t, bot.Slug, response.Bots[0].Slug)
		assert.Equal(t, bot.Name, response.Bots[0].Name)
	})
}

func TestBotHandler_Attack(t *testing.T) {
	handler, db, _ := setupBotHandlerTest(t)

	setupAttackTestData := func() (*domain.Location, *domain.User, *domain.Bot, error) {
		location := &domain.Location{
			Name:     fmt.Sprintf("Test Location %d", time.Now().UnixNano()),
			Slug:     fmt.Sprintf("test-location-%d", time.Now().UnixNano()),
			Cell:     false,
			Inactive: false,
		}
		locationRepo := repository.NewLocationRepository(db)
		err := locationRepo.Create(location)
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
		err = userRepo.Create(user)
		if err != nil {
			return nil, nil, nil, err
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
		err = botRepo.Create(bot)
		if err != nil {
			return nil, nil, nil, err
		}

		linkID := uuid.New()
		linkQuery := `INSERT INTO location_bots (id, location_id, bot_id) VALUES ($1, $2, $3)`
		_, err = db.Exec(linkQuery, linkID, location.ID, bot.ID)
		if err != nil {
			return nil, nil, nil, err
		}

		return location, user, bot, nil
	}

	t.Run("empty bot slug returns bad request", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.SetPath("/api/bots/:slug/attack")
		c.SetParamNames("slug")
		c.SetParamValues("")

		userID := uuid.New()
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		c.SetRequest(req.WithContext(ctx))

		err := handler.Attack(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "bot slug is required")
	})

	t.Run("unauthorized without userID in context returns unauthorized", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.SetPath("/api/bots/:slug/attack")
		c.SetParamNames("slug")
		c.SetParamValues("test-bot")

		err := handler.Attack(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("non-existent bot returns internal server error", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.SetPath("/api/bots/:slug/attack")
		c.SetParamNames("slug")
		c.SetParamValues("non-existent-bot")

		userID := uuid.New()
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		c.SetRequest(req.WithContext(ctx))

		err := handler.Attack(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("successfully attack bot", func(t *testing.T) {
		_, user, bot, err := setupAttackTestData()
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.SetPath("/api/bots/:slug/attack")
		c.SetParamNames("slug")
		c.SetParamValues(bot.Slug)

		ctx := context.WithValue(req.Context(), middleware.UserIDKey, user.ID)
		c.SetRequest(req.WithContext(ctx))

		err = handler.Attack(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["message"])

		fightRepo := repository.NewFightRepository(db)
		fight, err := fightRepo.FindActiveByUserID(user.ID)
		require.NoError(t, err)
		require.NotNil(t, fight)
		assert.Equal(t, user.ID, fight.UserID)
		assert.Equal(t, bot.ID, fight.BotID)
		assert.Equal(t, domain.FightStatusInProgress, fight.Status)

		var round domain.Round
		roundQuery := `
			SELECT id, created_at, deleted_at, fight_id, player_damage, bot_damage, 
				status, player_hp, bot_hp, player_attack_point, player_defense_point, 
				bot_attack_point, bot_defense_point
			FROM rounds 
			WHERE fight_id = $1 AND deleted_at IS NULL 
			ORDER BY created_at DESC 
			LIMIT 1
		`
		err = db.Get(&round, roundQuery, fight.ID)
		require.NoError(t, err)
		assert.Equal(t, fight.ID, round.FightID)
		assert.Equal(t, uint(0), round.PlayerDamage, "player damage should be 0")
		assert.Equal(t, uint(0), round.BotDamage, "bot damage should be 0")
		assert.Equal(t, user.CurrentHp, round.PlayerHp, "player HP should match user current HP")
		assert.Equal(t, bot.Hp, round.BotHp, "bot HP should match bot max HP")
		assert.Equal(t, domain.FightStatusInProgress, round.Status)
	})
}

func TestBotResponse_Marshal(t *testing.T) {
	bots := []*dto.Bot{
		{
			ID:      "123",
			Name:    "Test Bot",
			Slug:    "test-bot",
			Attack:  5,
			Defense: 3,
			Hp:      20,
			Level:   1,
			Avatar:  "images/bots/test",
		},
	}

	response := BotResponse{Bots: bots}
	data, err := json.Marshal(response)
	require.NoError(t, err)

	var unmarshalled BotResponse
	err = json.Unmarshal(data, &unmarshalled)
	require.NoError(t, err)
	assert.Equal(t, len(bots), len(unmarshalled.Bots))
	assert.Equal(t, bots[0].ID, unmarshalled.Bots[0].ID)
}
