package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
	"moonshine/internal/util"
)

const testJWTKey = "test-jwt-secret-key"

func setupAuthTestData(db *sqlx.DB) error {
	locationRepo := repository.NewLocationRepository(db)

	_, err := locationRepo.FindBySlug("moonshine")
	if err == nil {
		return nil
	}

	location := &domain.Location{
		Name: "Moonshine",
		Slug: "moonshine",
		Cell: false,
	}
	return locationRepo.Create(location)
}

func TestAuthService_SignUp(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	err := setupAuthTestData(testDB)
	require.NoError(t, err, "failed to setup test data")

	db := testDB
	userRepo := repository.NewUserRepository(db)
	avatarRepo := repository.NewAvatarRepository(db)
	locationRepo := repository.NewLocationRepository(db)

	service := NewAuthService(userRepo, avatarRepo, locationRepo, testJWTKey)

	t.Run("successful signup", func(t *testing.T) {
		ts := time.Now().UnixNano()
		input := SignUpInput{
			Username: fmt.Sprintf("u%d", ts%1000000),
			Email:    fmt.Sprintf("u%d@test.com", ts),
			Password: "password123",
		}

		user, token, err := service.SignUp(context.Background(), input)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.NotEmpty(t, token)
		assert.Equal(t, input.Username, user.Username)
		assert.Equal(t, input.Email, user.Email)
		assert.Equal(t, uint(1), user.Attack)
		assert.Equal(t, uint(1), user.Defense)
		assert.Equal(t, uint(20), user.Hp)
		assert.Equal(t, 20, user.CurrentHp)
		assert.Equal(t, uint(100), user.Gold)

		// Verify token is valid and contains correct user ID
		parsed, err := jwtv5.Parse(token, func(t *jwtv5.Token) (interface{}, error) {
			return []byte(testJWTKey), nil
		})
		require.NoError(t, err)
		require.True(t, parsed.Valid)

		claims, ok := parsed.Claims.(jwtv5.MapClaims)
		require.True(t, ok)
		assert.Equal(t, user.ID.String(), claims["id"])
	})

	t.Run("duplicate username returns error", func(t *testing.T) {
		ts := time.Now().UnixNano()
		input := SignUpInput{
			Username: fmt.Sprintf("d%d", ts%1000000),
			Email:    fmt.Sprintf("dup%d@test.com", ts),
			Password: "password123",
		}

		_, _, err := service.SignUp(context.Background(), input)
		require.NoError(t, err)

		input2 := SignUpInput{
			Username: input.Username,
			Email:    fmt.Sprintf("dup2_%d@test.com", ts),
			Password: "password123",
		}

		_, _, err = service.SignUp(context.Background(), input2)
		assert.ErrorIs(t, err, ErrUserAlreadyExists)
	})

	t.Run("invalid input - short username", func(t *testing.T) {
		input := SignUpInput{
			Username: "ab",
			Email:    "test@test.com",
			Password: "password123",
		}

		_, _, err := service.SignUp(context.Background(), input)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("invalid input - bad email", func(t *testing.T) {
		input := SignUpInput{
			Username: "validuser",
			Email:    "not-an-email",
			Password: "password123",
		}

		_, _, err := service.SignUp(context.Background(), input)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("invalid input - short password", func(t *testing.T) {
		input := SignUpInput{
			Username: "validuser",
			Email:    "valid@test.com",
			Password: "ab",
		}

		_, _, err := service.SignUp(context.Background(), input)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("invalid input - empty fields", func(t *testing.T) {
		input := SignUpInput{}

		_, _, err := service.SignUp(context.Background(), input)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})
}

func TestAuthService_SignIn(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	err := setupAuthTestData(testDB)
	require.NoError(t, err, "failed to setup test data")

	db := testDB
	userRepo := repository.NewUserRepository(db)
	avatarRepo := repository.NewAvatarRepository(db)
	locationRepo := repository.NewLocationRepository(db)

	service := NewAuthService(userRepo, avatarRepo, locationRepo, testJWTKey)

	// Create a user to sign in with
	ts := time.Now().UnixNano()
	password := "password123"
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	location, err := locationRepo.FindStartLocation()
	require.NoError(t, err)

	user := &domain.User{
		Username:   fmt.Sprintf("s%d", ts%1000000),
		Email:      fmt.Sprintf("signin%d@test.com", ts),
		Password:   hashedPassword,
		LocationID: location.ID,
		Attack:     1,
		Defense:    1,
		Hp:         20,
		CurrentHp:  20,
		Level:      1,
	}
	require.NoError(t, userRepo.Create(user))

	t.Run("successful signin", func(t *testing.T) {
		input := SignInInput{
			Username: user.Username,
			Password: password,
		}

		result, token, err := service.SignIn(context.Background(), input)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, token)
		assert.Equal(t, user.ID, result.ID)
		assert.Equal(t, user.Username, result.Username)

		// Verify token is valid and contains correct user ID
		parsed, err := jwtv5.Parse(token, func(t *jwtv5.Token) (interface{}, error) {
			return []byte(testJWTKey), nil
		})
		require.NoError(t, err)
		require.True(t, parsed.Valid)

		claims, ok := parsed.Claims.(jwtv5.MapClaims)
		require.True(t, ok)
		assert.Equal(t, user.ID.String(), claims["id"])
	})

	t.Run("wrong password", func(t *testing.T) {
		input := SignInInput{
			Username: user.Username,
			Password: "wrong-password",
		}

		_, _, err := service.SignIn(context.Background(), input)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("non-existent user", func(t *testing.T) {
		input := SignInInput{
			Username: "nonexistent",
			Password: "password123",
		}

		_, _, err := service.SignIn(context.Background(), input)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("invalid input - short username", func(t *testing.T) {
		input := SignInInput{
			Username: "ab",
			Password: "password123",
		}

		_, _, err := service.SignIn(context.Background(), input)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("invalid input - empty fields", func(t *testing.T) {
		input := SignInInput{}

		_, _, err := service.SignIn(context.Background(), input)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})
}

func TestAuthService_JWTTokenGeneration(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	err := setupAuthTestData(testDB)
	require.NoError(t, err, "failed to setup test data")

	db := testDB
	userRepo := repository.NewUserRepository(db)
	avatarRepo := repository.NewAvatarRepository(db)
	locationRepo := repository.NewLocationRepository(db)

	t.Run("token uses configured key", func(t *testing.T) {
		service := NewAuthService(userRepo, avatarRepo, locationRepo, "my-secret-key")

		ts := time.Now().UnixNano()
		input := SignUpInput{
			Username: fmt.Sprintf("j%d", ts%1000000),
			Email:    fmt.Sprintf("jwt%d@test.com", ts),
			Password: "password123",
		}

		_, token, err := service.SignUp(context.Background(), input)
		require.NoError(t, err)

		// Should parse with correct key
		parsed, err := jwtv5.Parse(token, func(t *jwtv5.Token) (interface{}, error) {
			return []byte("my-secret-key"), nil
		})
		require.NoError(t, err)
		assert.True(t, parsed.Valid)

		// Should fail with wrong key
		_, err = jwtv5.Parse(token, func(t *jwtv5.Token) (interface{}, error) {
			return []byte("wrong-key"), nil
		})
		assert.Error(t, err)
	})

	t.Run("token has expiry claim", func(t *testing.T) {
		service := NewAuthService(userRepo, avatarRepo, locationRepo, testJWTKey)

		ts := time.Now().UnixNano()
		input := SignUpInput{
			Username: fmt.Sprintf("e%d", ts%1000000),
			Email:    fmt.Sprintf("exp%d@test.com", ts),
			Password: "password123",
		}

		_, token, err := service.SignUp(context.Background(), input)
		require.NoError(t, err)

		parsed, err := jwtv5.Parse(token, func(t *jwtv5.Token) (interface{}, error) {
			return []byte(testJWTKey), nil
		})
		require.NoError(t, err)

		claims, ok := parsed.Claims.(jwtv5.MapClaims)
		require.True(t, ok)

		exp, err := claims.GetExpirationTime()
		require.NoError(t, err)
		require.NotNil(t, exp)
		// Token should expire roughly 72 hours from now
		assert.WithinDuration(t, time.Now().Add(72*time.Hour), exp.Time, 5*time.Second)
	})

	t.Run("token uses HS256 signing method", func(t *testing.T) {
		service := NewAuthService(userRepo, avatarRepo, locationRepo, testJWTKey)

		ts := time.Now().UnixNano()
		input := SignUpInput{
			Username: fmt.Sprintf("a%d", ts%1000000),
			Email:    fmt.Sprintf("alg%d@test.com", ts),
			Password: "password123",
		}

		_, token, err := service.SignUp(context.Background(), input)
		require.NoError(t, err)

		parsed, err := jwtv5.Parse(token, func(tok *jwtv5.Token) (interface{}, error) {
			assert.Equal(t, jwtv5.SigningMethodHS256, tok.Method)
			return []byte(testJWTKey), nil
		})
		require.NoError(t, err)
		assert.True(t, parsed.Valid)
	})
}
