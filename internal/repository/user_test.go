package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/domain"
)

func TestUserRepository_Create(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	repo := NewUserRepository(testDB)
	locationRepo := NewLocationRepository(testDB)
	ts := time.Now().UnixNano()

	location := &domain.Location{
		Name:     fmt.Sprintf("Test Location %d", ts),
		Slug:     fmt.Sprintf("test-location-%d", ts),
		Cell:     false,
		Inactive: false,
	}
	err := locationRepo.Create(location)
	require.NoError(t, err)

	user := &domain.User{
		Username:   fmt.Sprintf("testuser%d", ts),
		Email:      fmt.Sprintf("test%d@example.com", ts),
		Password:   "hashedpassword",
		LocationID: location.ID,
	}

	err = repo.Create(user)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)
}

func TestUserRepository_FindByUsername(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	repo := NewUserRepository(testDB)
	locationRepo := NewLocationRepository(testDB)
	ts := time.Now().UnixNano()

	location := &domain.Location{
		Name:     fmt.Sprintf("Test Location %d", ts),
		Slug:     fmt.Sprintf("test-location-%d", ts),
		Cell:     false,
		Inactive: false,
	}
	err := locationRepo.Create(location)
	require.NoError(t, err)

	username := fmt.Sprintf("finduser%d", ts)
	user := &domain.User{
		Username:   username,
		Email:      fmt.Sprintf("find%d@example.com", ts),
		Password:   "hashedpassword",
		LocationID: location.ID,
	}
	err = repo.Create(user)
	require.NoError(t, err)

	found, err := repo.FindByUsername(username)
	require.NoError(t, err)
	assert.Equal(t, username, found.Username)
}

func TestUserRepository_FindByID(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	repo := NewUserRepository(testDB)
	locationRepo := NewLocationRepository(testDB)
	ts := time.Now().UnixNano()

	location := &domain.Location{
		Name:     fmt.Sprintf("Test Location %d", ts),
		Slug:     fmt.Sprintf("test-location-%d", ts),
		Cell:     false,
		Inactive: false,
	}
	err := locationRepo.Create(location)
	require.NoError(t, err)

	user := &domain.User{
		Username:   fmt.Sprintf("iduser%d", ts),
		Email:      fmt.Sprintf("id%d@example.com", ts),
		Password:   "hashedpassword",
		LocationID: location.ID,
	}
	err = repo.Create(user)
	require.NoError(t, err)

	found, err := repo.FindByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
}

func TestUserRepository_FindByUsername_NotFound(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	repo := NewUserRepository(testDB)

	_, err := repo.FindByUsername("nonexistent")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	repo := NewUserRepository(testDB)

	_, err := repo.FindByID(uuid.New())
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepository_RegenerateAllUsersHealth(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	repo := NewUserRepository(testDB)
	locationRepo := NewLocationRepository(testDB)
	botRepo := NewBotRepository(testDB)
	fightRepo := NewFightRepository(testDB)
	ts := time.Now().UnixNano()

	location := &domain.Location{
		Name:     fmt.Sprintf("Test Location %d", ts),
		Slug:     fmt.Sprintf("test-location-%d", ts),
		Cell:     false,
		Inactive: false,
	}
	err := locationRepo.Create(location)
	require.NoError(t, err)

	bot := &domain.Bot{
		Name:    "Test Bot",
		Slug:    fmt.Sprintf("test-bot-%d", ts),
		Attack:  5,
		Defense: 3,
		Hp:      20,
		Level:   1,
		Avatar:  "images/bots/test",
	}
	err = botRepo.Create(bot)
	require.NoError(t, err)

	userInFight := &domain.User{
		Username:   fmt.Sprintf("infight%d", ts),
		Email:      fmt.Sprintf("infight%d@example.com", ts),
		Password:   "hashedpassword",
		LocationID: location.ID,
		Hp:         100,
		CurrentHp:  50,
		Level:      1,
	}
	err = repo.Create(userInFight)
	require.NoError(t, err)

	userNotInFight := &domain.User{
		Username:   fmt.Sprintf("notinfight%d", ts),
		Email:      fmt.Sprintf("notinfight%d@example.com", ts),
		Password:   "hashedpassword",
		LocationID: location.ID,
		Hp:         100,
		CurrentHp:  50,
		Level:      1,
	}
	err = repo.Create(userNotInFight)
	require.NoError(t, err)

	fight := &domain.Fight{
		UserID: userInFight.ID,
		BotID:  bot.ID,
		Status: domain.FightStatusInProgress,
	}
	_, err = fightRepo.Create(fight)
	require.NoError(t, err)

	initialHpInFight := userInFight.CurrentHp
	initialHpNotInFight := userNotInFight.CurrentHp

	_, err = repo.RegenerateAllUsersHealth(10.0)
	require.NoError(t, err)

	userInFightAfter, err := repo.FindByID(userInFight.ID)
	require.NoError(t, err)
	assert.Equal(t, initialHpInFight, userInFightAfter.CurrentHp, "HP should not regenerate for user in fight")

	userNotInFightAfter, err := repo.FindByID(userNotInFight.ID)
	require.NoError(t, err)
	assert.Greater(t, userNotInFightAfter.CurrentHp, initialHpNotInFight, "HP should regenerate for user not in fight")
}
