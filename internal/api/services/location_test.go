package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

func setupLocationTestData(db *sqlx.DB) ([]*domain.Location, error) {
	locations := []*domain.Location{
		{Name: "Cell 1", Slug: "1cell", Cell: true, Inactive: false},
		{Name: "Cell 2", Slug: "2cell", Cell: true, Inactive: false},
		{Name: "Cell 3", Slug: "3cell", Cell: true, Inactive: false},
		{Name: "Cell 4", Slug: "4cell", Cell: true, Inactive: false},
		{Name: "Cell 5", Slug: "5cell", Cell: true, Inactive: false},
	}

	locationRepo := repository.NewLocationRepository(db)
	for _, loc := range locations {
		if err := locationRepo.Create(loc); err != nil {
			return nil, err
		}
	}

	return locations, nil
}

func createConnections(db *sqlx.DB, connections [][2]uuid.UUID) error {
	for _, conn := range connections {
		query := `INSERT INTO location_locations (id, location_id, near_location_id) VALUES ($1, $2, $3)`
		_, err := db.Exec(query, uuid.New(), conn[0], conn[1])
		if err != nil {
			return err
		}
	}
	return nil
}

type noopMovingWorker struct{}

func (noopMovingWorker) StartMovement(userID uuid.UUID, cellSlugs []string) error { return nil }

func TestLocationService_FindShortestPath(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}

	db := testDB

	t.Run("successful path finding - direct connection", func(t *testing.T) {
		db.Exec("TRUNCATE TABLE location_locations CASCADE")
		db.Exec("TRUNCATE TABLE locations CASCADE")

		locations, err := setupLocationTestData(db)
		require.NoError(t, err)

		err = createConnections(db, [][2]uuid.UUID{
			{locations[0].ID, locations[1].ID},
		})
		require.NoError(t, err)

		locationRepo := repository.NewLocationRepository(db)
		userRepo := repository.NewUserRepository(db)
		service, err := NewLocationService(db, nil, locationRepo, userRepo, noopMovingWorker{})
		require.NoError(t, err)

		path, err := service.FindShortestPath(locations[0].Slug, locations[1].Slug)
		require.NoError(t, err)
		assert.Len(t, path, 1)
		assert.Equal(t, locations[1].Slug, path[0])
	})

	t.Run("successful path finding - multi-step path", func(t *testing.T) {
		db.Exec("TRUNCATE TABLE location_locations CASCADE")
		db.Exec("TRUNCATE TABLE locations CASCADE")

		locations, err := setupLocationTestData(db)
		require.NoError(t, err)

		err = createConnections(db, [][2]uuid.UUID{
			{locations[0].ID, locations[1].ID},
			{locations[1].ID, locations[2].ID},
			{locations[2].ID, locations[3].ID},
		})
		require.NoError(t, err)

		locationRepo := repository.NewLocationRepository(db)
		userRepo := repository.NewUserRepository(db)
		service, err := NewLocationService(db, nil, locationRepo, userRepo, noopMovingWorker{})
		require.NoError(t, err)

		path, err := service.FindShortestPath(locations[0].Slug, locations[3].Slug)
		require.NoError(t, err)

		expectedPath := []string{locations[1].Slug, locations[2].Slug, locations[3].Slug}
		assert.Equal(t, expectedPath, path)
	})

	t.Run("same start and end location", func(t *testing.T) {
		db.Exec("TRUNCATE TABLE location_locations CASCADE")
		db.Exec("TRUNCATE TABLE locations CASCADE")

		locations, err := setupLocationTestData(db)
		require.NoError(t, err)

		locationRepo := repository.NewLocationRepository(db)
		userRepo := repository.NewUserRepository(db)
		service, err := NewLocationService(db, nil, locationRepo, userRepo, noopMovingWorker{})
		require.NoError(t, err)

		path, err := service.FindShortestPath(locations[0].Slug, locations[0].Slug)
		require.NoError(t, err)
		assert.Len(t, path, 0)
	})

	t.Run("path not found - disconnected locations", func(t *testing.T) {
		db.Exec("TRUNCATE TABLE location_locations CASCADE")
		db.Exec("TRUNCATE TABLE locations CASCADE")

		locations, err := setupLocationTestData(db)
		require.NoError(t, err)

		err = createConnections(db, [][2]uuid.UUID{
			{locations[0].ID, locations[1].ID},
		})
		require.NoError(t, err)

		locationRepo := repository.NewLocationRepository(db)
		userRepo := repository.NewUserRepository(db)
		service, err := NewLocationService(db, nil, locationRepo, userRepo, noopMovingWorker{})
		require.NoError(t, err)

		_, err = service.FindShortestPath(locations[0].Slug, locations[4].Slug)
		assert.ErrorIs(t, err, ErrLocationNotConnected)
	})

	t.Run("non-existent start location", func(t *testing.T) {
		db.Exec("TRUNCATE TABLE location_locations CASCADE")
		db.Exec("TRUNCATE TABLE locations CASCADE")

		locations, err := setupLocationTestData(db)
		require.NoError(t, err)

		locationRepo := repository.NewLocationRepository(db)
		userRepo := repository.NewUserRepository(db)
		service, err := NewLocationService(db, nil, locationRepo, userRepo, noopMovingWorker{})
		require.NoError(t, err)

		_, err = service.FindShortestPath("non_existent", locations[0].Slug)
		assert.Error(t, err)
	})

	t.Run("non-existent end location", func(t *testing.T) {
		db.Exec("TRUNCATE TABLE location_locations CASCADE")
		db.Exec("TRUNCATE TABLE locations CASCADE")

		locations, err := setupLocationTestData(db)
		require.NoError(t, err)

		locationRepo := repository.NewLocationRepository(db)
		userRepo := repository.NewUserRepository(db)
		service, err := NewLocationService(db, nil, locationRepo, userRepo, noopMovingWorker{})
		require.NoError(t, err)

		_, err = service.FindShortestPath(locations[0].Slug, "non_existent")
		assert.Error(t, err)
	})

	t.Run("path with alternative routes - shortest path", func(t *testing.T) {
		db.Exec("TRUNCATE TABLE location_locations CASCADE")
		db.Exec("TRUNCATE TABLE locations CASCADE")

		locations, err := setupLocationTestData(db)
		require.NoError(t, err)

		err = createConnections(db, [][2]uuid.UUID{
			{locations[0].ID, locations[1].ID},
			{locations[1].ID, locations[4].ID},
			{locations[0].ID, locations[2].ID},
			{locations[2].ID, locations[3].ID},
			{locations[3].ID, locations[4].ID},
		})
		require.NoError(t, err)

		locationRepo := repository.NewLocationRepository(db)
		userRepo := repository.NewUserRepository(db)
		service, err := NewLocationService(db, nil, locationRepo, userRepo, noopMovingWorker{})
		require.NoError(t, err)

		path, err := service.FindShortestPath(locations[0].Slug, locations[4].Slug)
		require.NoError(t, err)

		expectedShortestPath := []string{locations[1].Slug, locations[4].Slug}
		assert.Equal(t, expectedShortestPath, path)
	})
}
