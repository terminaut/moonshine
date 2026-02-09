package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/repository"
	"moonshine/internal/worker"
)

type LocationHandler struct {
	db              *sqlx.DB
	locationService *services.LocationService
	locationRepo    *repository.LocationRepository
	userRepo        *repository.UserRepository
}

type LocationCellsResponse struct {
	Cells []locationCell `json:"cells"`
}

type MoveToCellResponse struct {
	Message      string `json:"message"`
	PathLength   int    `json:"path_length"`
	TargetCell   string `json:"target_cell"`
	TimePerCell  int    `json:"time_per_cell"`
}

type locationCell struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Image    string `json:"image"`
	Inactive bool   `json:"inactive"`
}


func NewLocationHandler(db *sqlx.DB, rdb *redis.Client) *LocationHandler {
	locationRepo := repository.NewLocationRepository(db)
	userRepo := repository.NewUserRepository(db)
	movingWorker := worker.NewCellsMovingWorker(locationRepo, userRepo, rdb, 5*time.Second)
	locationService, err := services.NewLocationService(db, rdb, locationRepo, userRepo, movingWorker)
	if err != nil {
		log.Fatalf("Failed to create LocationService: %v", err)
	}

	return &LocationHandler{
		db:              db,
		locationService: locationService,
		locationRepo:    locationRepo,
		userRepo:        userRepo,
	}
}

// MoveToLocation godoc
// @Summary Move to location
// @Description Move user to a different location
// @Tags locations
// @Accept json
// @Produce json
// @Security Bearer
// @Param slug path string true "Location slug"
// @Success 200
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/locations/{slug}/move [post]
func (h *LocationHandler) MoveToLocation(c echo.Context) error {
	locationSlug := c.Param("slug")
	if locationSlug == "" {
		return ErrBadRequest(c, "location slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
		return err
	}

	err = h.locationService.MoveToLocation(c.Request().Context(), userID, locationSlug)
	if err != nil {
		switch err {
		case services.ErrLocationNotConnected:
			return ErrBadRequest(c, "locations not connected")
		case repository.ErrLocationNotFound:
			return ErrNotFound(c, "location not found")
		case repository.ErrUserNotFound:
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	return c.JSON(http.StatusOK, nil)
}

// MoveToCell godoc
// @Summary Move to cell
// @Description Start movement to a cell within location
// @Tags locations
// @Accept json
// @Produce json
// @Security Bearer
// @Param slug path string true "Location slug"
// @Param cell_slug path string true "Cell slug"
// @Success 200 {object} MoveToCellResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/locations/{slug}/cells/{cell_slug}/move [post]
func (h *LocationHandler) MoveToCell(c echo.Context) error {
	cellSlug := c.Param("cell_slug")
	if cellSlug == "" {
		return ErrBadRequest(c, "cell slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
		return err
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		return ErrNotFound(c, "user not found")
	}

	currentLocation, err := h.locationRepo.FindByID(user.LocationID)
	if err != nil {
		return ErrNotFound(c, "location not found")
	}

	if currentLocation.Slug == cellSlug {
		return c.JSON(http.StatusOK, nil)
	}

	path, err := h.locationService.FindShortestPath(currentLocation.Slug, cellSlug)
	if err != nil {
		switch err {
		case services.ErrLocationNotConnected:
			return ErrBadRequest(c, "locations not connected")
		case repository.ErrLocationNotFound:
			return ErrNotFound(c, "location not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	if err := h.locationService.StartCellMovement(userID, path); err != nil {
		return ErrBadRequest(c, "")
	}

	targetLocation, err := h.locationRepo.FindBySlug(cellSlug)
	targetName := cellSlug
	if err == nil && targetLocation != nil {
		targetName = targetLocation.Name
	} else {
		targetName = strings.TrimSuffix(cellSlug, "cell")
	}

	return c.JSON(http.StatusOK, &MoveToCellResponse{
		Message:     "movement started",
		PathLength:  len(path),
		TargetCell:  targetName,
		TimePerCell: 5,
	})
}

// GetLocationCells godoc
// @Summary Get location cells
// @Description Get list of cells in a location
// @Tags locations
// @Accept json
// @Produce json
// @Security Bearer
// @Param slug path string true "Location slug"
// @Success 200 {object} LocationCellsResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/locations/{slug}/cells [get]
func (h *LocationHandler) GetLocationCells(c echo.Context) error {
	locationSlug := c.Param("slug")
	if locationSlug == "" {
		return ErrBadRequest(c, "location slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
		return err
	}

	locationRepo := repository.NewLocationRepository(h.db)
	location, err := locationRepo.FindBySlug(locationSlug)
	if err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			return ErrNotFound(c, "location not found")
		}
		return ErrInternalServerError(c)
	}

	cells, err := locationRepo.FindCellsByLocationID(location.ID)
	if err != nil {
		return ErrInternalServerError(c)
	}

	cellsList := make([]locationCell, len(cells))
	for i, cell := range cells {
		cellsList[i] = locationCell{
			ID:       cell.ID.String(),
			Slug:     cell.Slug,
			Name:     cell.Name,
			Image:    cell.Image,
			Inactive: cell.Inactive,
		}
	}

	return c.JSON(http.StatusOK, &LocationCellsResponse{
		Cells: cellsList,
	})
}
