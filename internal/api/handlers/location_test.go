package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/api/middleware"
	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

func setupLocationHandlerTest(t *testing.T) (*LocationHandler, *sqlx.DB, *domain.User, *domain.Location, echo.Echo) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}
	db := testDB.DB()
	handler := NewLocationHandler(db, nil)
	loc := &domain.Location{
		Name:     fmt.Sprintf("Loc %d", time.Now().UnixNano()),
		Slug:     fmt.Sprintf("loc-%d", time.Now().UnixNano()),
		Cell:     false,
		Inactive: false,
	}
	locRepo := repository.NewLocationRepository(db)
	require.NoError(t, locRepo.Create(loc))
	user := &domain.User{
		Username:   fmt.Sprintf("u%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("u%d@x.com", time.Now().UnixNano()),
		Password:   "x",
		Name:       "User",
		LocationID: loc.ID,
		Attack:     1, Defense: 1, Hp: 20, CurrentHp: 20, Level: 1, Gold: 0, Exp: 0, FreeStats: 0,
	}
	userRepo := repository.NewUserRepository(db)
	require.NoError(t, userRepo.Create(user))
	e := echo.New()
	return handler, db, user, loc, *e
}

func TestLocationHandler_MoveToLocation(t *testing.T) {
	handler, _, user, loc, e := setupLocationHandlerTest(t)

	t.Run("empty slug returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/locations//move", nil)
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/locations/:slug/move")
		c.SetParamNames("slug")
		c.SetParamValues("")

		err := handler.MoveToLocation(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unauthorized when no userID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/locations/any/move", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/locations/:slug/move")
		c.SetParamNames("slug")
		c.SetParamValues("any")

		err := handler.MoveToLocation(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("location not found returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/locations/nonexistent/move", nil)
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/locations/:slug/move")
		c.SetParamNames("slug")
		c.SetParamValues("nonexistent")

		err := handler.MoveToLocation(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("move to same location returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/locations/"+loc.Slug+"/move", nil)
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/locations/:slug/move")
		c.SetParamNames("slug")
		c.SetParamValues(loc.Slug)

		err := handler.MoveToLocation(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestLocationHandler_GetLocationCells(t *testing.T) {
	handler, db, user, loc, e := setupLocationHandlerTest(t)

	cell := &domain.Location{Name: "C1", Slug: fmt.Sprintf("1cell-%d", time.Now().UnixNano()), Cell: true, Inactive: false}
	locRepo := repository.NewLocationRepository(db)
	require.NoError(t, locRepo.Create(cell))

	t.Run("empty slug returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/locations//cells", nil)
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/locations/:slug/cells")
		c.SetParamNames("slug")
		c.SetParamValues("")

		err := handler.GetLocationCells(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unauthorized when no userID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/locations/x/cells", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/locations/:slug/cells")
		c.SetParamNames("slug")
		c.SetParamValues("x")

		err := handler.GetLocationCells(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("location not found returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/locations/nonexistent/cells", nil)
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/locations/:slug/cells")
		c.SetParamNames("slug")
		c.SetParamValues("nonexistent")

		err := handler.GetLocationCells(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("success returns 200 and cells", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/locations/"+loc.Slug+"/cells", nil)
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/locations/:slug/cells")
		c.SetParamNames("slug")
		c.SetParamValues(loc.Slug)

		err := handler.GetLocationCells(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp LocationCellsResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Cells)
	})
}

func TestLocationHandler_MoveToCell(t *testing.T) {
	handler, _, user, loc, e := setupLocationHandlerTest(t)

	t.Run("empty cell_slug returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/locations/x/cells//move", nil)
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/locations/:slug/cells/:cell_slug/move")
		c.SetParamNames("slug", "cell_slug")
		c.SetParamValues(loc.Slug, "")

		err := handler.MoveToCell(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unauthorized when no userID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/locations/x/cells/y/move", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/locations/:slug/cells/:cell_slug/move")
		c.SetParamNames("slug", "cell_slug")
		c.SetParamValues("x", "y")

		err := handler.MoveToCell(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
