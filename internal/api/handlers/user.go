package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/repository"
)

type UserHandler struct {
	db               *sqlx.DB
	rdb              *redis.Client
	userService      *services.UserService
	inventoryService *services.InventoryService
	userRepo         *repository.UserRepository
}

func NewUserHandler(db *sqlx.DB, rdb *redis.Client) *UserHandler {
	userRepo := repository.NewUserRepository(db)
	avatarRepo := repository.NewAvatarRepository(db)
	locationRepo := repository.NewLocationRepository(db)
	userService := services.NewUserService(userRepo, avatarRepo, locationRepo)

	inventoryRepo := repository.NewInventoryRepository(db)
	inventoryService := services.NewInventoryService(inventoryRepo)

	return &UserHandler{
		db:               db,
		rdb:              rdb,
		userService:      userService,
		inventoryService: inventoryService,
		userRepo:         userRepo,
	}
}

// GetCurrentUser godoc
// @Summary Get current user
// @Description Get authenticated user information
// @Tags user
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} dto.User
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/user/me [get]
func (h *UserHandler) GetCurrentUser(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	user, location, inFight, err := h.userService.GetCurrentUserWithRelations(c.Request().Context(), userID, h.rdb)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return ErrNotFound(c, "user not found")
		}
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, dto.UserFromDomain(user, location, nil, inFight))
}

// GetUserInventory godoc
// @Summary Get user inventory
// @Description Get list of items in user's inventory
// @Tags user
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {array} dto.EquipmentItem
// @Failure 401 {object} map[string]string
// @Router /api/users/me/inventory [get]
func (h *UserHandler) GetUserInventory(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	items, err := h.inventoryService.GetUserInventory(c.Request().Context(), userID)
	if err != nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, dto.EquipmentItemsFromDomain(items))
}

// GetUserEquippedItems godoc
// @Summary Get equipped items
// @Description Get list of currently equipped items
// @Tags user
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} map[string]dto.EquipmentItem
// @Failure 401 {object} map[string]string
// @Router /api/users/me/equipped [get]
func (h *UserHandler) GetUserEquippedItems(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	userRepo := repository.NewUserRepository(h.db)
	user, err := userRepo.FindByID(userID)
	if err != nil {
		return ErrNotFound(c, "user not found")
	}

	slots := []struct {
		name string
		id   *uuid.UUID
	}{
		{"chest", user.ChestEquipmentItemID},
		{"belt", user.BeltEquipmentItemID},
		{"head", user.HeadEquipmentItemID},
		{"neck", user.NeckEquipmentItemID},
		{"weapon", user.WeaponEquipmentItemID},
		{"shield", user.ShieldEquipmentItemID},
		{"legs", user.LegsEquipmentItemID},
		{"feet", user.FeetEquipmentItemID},
		{"arms", user.ArmsEquipmentItemID},
		{"hands", user.HandsEquipmentItemID},
		{"ring1", user.Ring1EquipmentItemID},
		{"ring2", user.Ring2EquipmentItemID},
		{"ring3", user.Ring3EquipmentItemID},
		{"ring4", user.Ring4EquipmentItemID},
	}
	var ids []uuid.UUID
	for _, s := range slots {
		if s.id != nil {
			ids = append(ids, *s.id)
		}
	}
	if len(ids) == 0 {
		return c.JSON(http.StatusOK, map[string]*dto.EquipmentItem{})
	}

	equipmentItemRepo := repository.NewEquipmentItemRepository(h.db)
	list, err := equipmentItemRepo.FindByIDs(ids)
	if err != nil {
		return ErrInternalServerError(c)
	}
	idToItem := make(map[uuid.UUID]*dto.EquipmentItem)
	for _, it := range list {
		idToItem[it.ID] = dto.EquipmentItemFromDomain(it)
	}

	equipmentItems := map[string]*dto.EquipmentItem{}
	for _, s := range slots {
		if s.id != nil {
			if d, ok := idToItem[*s.id]; ok {
				equipmentItems[s.name] = d
			}
		}
	}
	return c.JSON(http.StatusOK, equipmentItems)
}

// UpdateCurrentUser godoc
// @Summary Update current user
// @Description Update authenticated user information
// @Tags user
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.UpdateUserRequest true "Update user request"
// @Success 200 {object} dto.User
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/user/me [put]
func (h *UserHandler) UpdateCurrentUser(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
		return err
	}

	var req dto.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return ErrBadRequest(c, "invalid request")
	}

	var avatarID *uuid.UUID
	if req.AvatarID != nil {
		parsedID, err := uuid.Parse(*req.AvatarID)
		if err != nil {
			return ErrBadRequest(c, "invalid avatar ID")
		}
		avatarID = &parsedID
	}

	user, err := h.userService.UpdateUser(c.Request().Context(), userID, avatarID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return ErrNotFound(c, "user not found")
		}
		if err == repository.ErrAvatarNotFound {
			return ErrNotFound(c, "avatar not found")
		}
		return ErrInternalServerError(c)
	}

	user, location, inFight, err := h.userService.GetCurrentUserWithRelations(c.Request().Context(), userID, h.rdb)
	if err != nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, dto.UserFromDomain(user, location, nil, inFight))
}
