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

func TestBotRepository_Create(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	repo := NewBotRepository(testDB.DB())
	ts := time.Now().UnixNano()

	bot := &domain.Bot{
		Name:    fmt.Sprintf("Test Bot %d", ts),
		Slug:    fmt.Sprintf("test-bot-%d", ts),
		Attack:  5,
		Defense: 3,
		Hp:      20,
		Level:   1,
		Avatar:  "images/bots/test",
	}

	err := repo.Create(bot)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, bot.ID)
}

func TestBotRepository_FindBySlug(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	repo := NewBotRepository(testDB.DB())
	ts := time.Now().UnixNano()
	slug := fmt.Sprintf("find-bot-%d", ts)

	bot := &domain.Bot{
		Name:    fmt.Sprintf("Find Bot %d", ts),
		Slug:    slug,
		Attack:  5,
		Defense: 3,
		Hp:      20,
		Level:   1,
		Avatar:  "images/bots/find",
	}
	err := repo.Create(bot)
	require.NoError(t, err)

	found, err := repo.FindBySlug(slug)
	require.NoError(t, err)
	assert.Equal(t, slug, found.Slug)
	assert.Equal(t, bot.Name, found.Name)
}

func TestBotRepository_FindBySlug_NotFound(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	repo := NewBotRepository(testDB.DB())

	_, err := repo.FindBySlug("non-existent-slug")
	assert.ErrorIs(t, err, ErrBotNotFound)
}

func TestBotRepository_FindBotsByLocationID(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	repo := NewBotRepository(testDB.DB())
	locationRepo := NewLocationRepository(testDB.DB())
	ts := time.Now().UnixNano()

	location := &domain.Location{
		Name: fmt.Sprintf("Test Location %d", ts),
		Slug: fmt.Sprintf("test-location-%d", ts),
		Cell: false,
	}
	err := locationRepo.Create(location)
	require.NoError(t, err)

	bot1 := &domain.Bot{
		Name:    fmt.Sprintf("Bot 1 %d", ts),
		Slug:    fmt.Sprintf("bot-1-%d", ts),
		Attack:  5,
		Defense: 3,
		Hp:      20,
		Level:   1,
		Avatar:  "images/bots/bot1",
	}
	err = repo.Create(bot1)
	require.NoError(t, err)

	bot2 := &domain.Bot{
		Name:    fmt.Sprintf("Bot 2 %d", ts),
		Slug:    fmt.Sprintf("bot-2-%d", ts),
		Attack:  7,
		Defense: 4,
		Hp:      25,
		Level:   2,
		Avatar:  "images/bots/bot2",
	}
	err = repo.Create(bot2)
	require.NoError(t, err)

	linkID1 := uuid.New()
	linkQuery := `INSERT INTO location_bots (id, location_id, bot_id) VALUES ($1, $2, $3)`
	_, err = testDB.DB().Exec(linkQuery, linkID1, location.ID, bot1.ID)
	require.NoError(t, err)

	linkID2 := uuid.New()
	_, err = testDB.DB().Exec(linkQuery, linkID2, location.ID, bot2.ID)
	require.NoError(t, err)

	bots, err := repo.FindBotsByLocationID(location.ID)
	require.NoError(t, err)
	assert.Len(t, bots, 2)

	botSlugs := make(map[string]bool)
	for _, bot := range bots {
		botSlugs[bot.Slug] = true
	}
	assert.True(t, botSlugs[bot1.Slug])
	assert.True(t, botSlugs[bot2.Slug])
}

func TestBotRepository_FindBotsByLocationID_Empty(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	repo := NewBotRepository(testDB.DB())
	locationRepo := NewLocationRepository(testDB.DB())
	ts := time.Now().UnixNano()

	location := &domain.Location{
		Name: fmt.Sprintf("Empty Location %d", ts),
		Slug: fmt.Sprintf("empty-location-%d", ts),
		Cell: false,
	}
	err := locationRepo.Create(location)
	require.NoError(t, err)

	bots, err := repo.FindBotsByLocationID(location.ID)
	require.NoError(t, err)
	assert.Empty(t, bots)
}
