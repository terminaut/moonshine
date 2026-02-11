package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type customValidator struct{ v *validator.Validate }

func (cv *customValidator) Validate(i interface{}) error { return cv.v.Struct(i) }

func setupAuthHandlerTest(t *testing.T) (*AuthHandler, *sqlx.DB, echo.Echo) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}
	db := testDB.DB()
	handler := NewAuthHandler(db, "test-secret")
	e := echo.New()
	e.Validator = &customValidator{v: validator.New()}
	return handler, db, *e
}

func ensureMoonshineLocation(db *sqlx.DB) error {
	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM locations WHERE slug = 'moonshine' AND deleted_at IS NULL`)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = db.Exec(`INSERT INTO locations (name, slug, cell, inactive) VALUES ($1, $2, $3, $4)`,
		"Moonshine", "moonshine", false, false)
	return err
}

func TestAuthHandler_SignUp(t *testing.T) {
	handler, db, e := setupAuthHandlerTest(t)
	err := ensureMoonshineLocation(db)
	require.NoError(t, err)

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.SignUp(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		body := map[string]string{"username": "ab", "email": "x@y.z", "password": "short"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.SignUp(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("successful signup returns 200 and token", func(t *testing.T) {
		u := fmt.Sprintf("user%d", time.Now().UnixNano())
		body := map[string]string{
			"username": u,
			"email":    fmt.Sprintf("%s@test.com", u),
			"password": "password123",
		}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.SignUp(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp AuthResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
		assert.NotNil(t, resp.User)
		assert.Equal(t, u, resp.User.Username)
	})

	t.Run("duplicate username returns 409", func(t *testing.T) {
		u := fmt.Sprintf("dup%d", time.Now().UnixNano())
		userRepo := repository.NewUserRepository(db)
		loc := &domain.Location{Name: "L", Slug: "loc-" + u, Cell: false, Inactive: false}
		locRepo := repository.NewLocationRepository(db)
		require.NoError(t, locRepo.Create(loc))
		user := &domain.User{
			Username:   u,
			Name:       u,
			Email:      u + "@x.com",
			Password:   "hashed",
			LocationID: loc.ID,
			Attack:     1, Defense: 1, Hp: 20, CurrentHp: 20, Level: 1, Gold: 0, Exp: 0, FreeStats: 0,
		}
		require.NoError(t, userRepo.Create(user))

		body := map[string]string{"username": u, "email": "other@test.com", "password": "password123"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.SignUp(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestAuthHandler_SignIn(t *testing.T) {
	handler, db, e := setupAuthHandlerTest(t)
	err := ensureMoonshineLocation(db)
	require.NoError(t, err)

	u := fmt.Sprintf("signin%d", time.Now().UnixNano())
	signUpBody := map[string]string{"username": u, "email": u + "@test.com", "password": "pass123"}
	b, _ := json.Marshal(signUpBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err = handler.SignUp(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.SignIn(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid credentials returns 401", func(t *testing.T) {
		body := map[string]string{"username": u, "password": "wrong"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.SignIn(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("successful signin returns 200 and token", func(t *testing.T) {
		body := map[string]string{"username": u, "password": "pass123"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.SignIn(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp AuthResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
		assert.NotNil(t, resp.User)
		assert.Equal(t, u, resp.User.Username)
	})
}
