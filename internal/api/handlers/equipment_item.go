package handlers

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	r "moonshine/internal/redis"
	"moonshine/internal/repository"
)

type EquipmentItemHandler struct {
	db                          *sqlx.DB
	rdb                         *redis.Client
	equipmentItemService        *services.EquipmentItemService
	equipmentItemBuyService     *services.EquipmentItemBuyService
	equipmentItemSellService    *services.EquipmentItemSellService
	equipmentItemTakeOnService  *services.EquipmentItemTakeOnService
	equipmentItemTakeOffService *services.EquipmentItemTakeOffService
	userRepo                    *repository.UserRepository
}

func NewEquipmentItemHandler(db *sqlx.DB, rdb ...*redis.Client) *EquipmentItemHandler {
	equipmentItemRepo := repository.NewEquipmentItemRepository(db)
	equipmentItemService := services.NewEquipmentItemService(equipmentItemRepo)

	inventoryRepo := repository.NewInventoryRepository(db)
	userRepo := repository.NewUserRepository(db)
	equipmentItemBuyService := services.NewEquipmentItemBuyService(db, equipmentItemRepo, inventoryRepo, userRepo)
	equipmentItemSellService := services.NewEquipmentItemSellService(db, equipmentItemRepo, inventoryRepo, userRepo)
	equipmentItemTakeOnService := services.NewEquipmentItemTakeOnService(db, equipmentItemRepo, inventoryRepo, userRepo)
	equipmentItemTakeOffService := services.NewEquipmentItemTakeOffService(db, equipmentItemRepo, inventoryRepo, userRepo)

	var redisClient *redis.Client
	if len(rdb) > 0 {
		redisClient = rdb[0]
	}

	return &EquipmentItemHandler{
		rdb:                         redisClient,
		db:                          db,
		equipmentItemService:        equipmentItemService,
		equipmentItemBuyService:     equipmentItemBuyService,
		equipmentItemSellService:    equipmentItemSellService,
		equipmentItemTakeOnService:  equipmentItemTakeOnService,
		equipmentItemTakeOffService: equipmentItemTakeOffService,
		userRepo:                    userRepo,
	}
}

func (h *EquipmentItemHandler) invalidateUserCache(ctx context.Context, userID string) {
	if h.rdb == nil {
		return
	}
	_ = r.UserCache(h.rdb).Delete(ctx, userID)
}

// GetEquipmentItems godoc
// @Summary Get equipment items
// @Description Get list of equipment items by category
// @Tags equipment
// @Accept json
// @Produce json
// @Security Bearer
// @Param category query string true "Equipment category"
// @Success 200 {array} dto.EquipmentItem
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/equipment_items [get]
func (h *EquipmentItemHandler) GetEquipmentItems(c echo.Context) error {
	category := c.QueryParam("category")
	if category == "" {
		return ErrBadRequest(c, "category parameter is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
		return err
	}

	artifact := c.QueryParam("artifact") == "true"

	items, err := h.equipmentItemService.GetByCategorySlug(c.Request().Context(), category, artifact)
	if err != nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, dto.EquipmentItemsFromDomain(items))
}

// BuyEquipmentItem godoc
// @Summary Buy equipment item
// @Description Purchase an equipment item
// @Tags equipment
// @Accept json
// @Produce json
// @Security Bearer
// @Param slug path string true "Item slug"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/equipment_items/{slug}/buy [post]
func (h *EquipmentItemHandler) BuyEquipmentItem(c echo.Context) error {
	itemSlug := c.Param("slug")
	if itemSlug == "" {
		return ErrBadRequest(c, "item slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
		return err
	}

	err = h.equipmentItemBuyService.BuyEquipmentItem(c.Request().Context(), userID, itemSlug)
	if err != nil {
		switch err {
		case services.ErrEquipmentItemNotFound:
			return ErrNotFound(c, "equipment item not found")
		case services.ErrInsufficientGold:
			return ErrBadRequest(c, "insufficient gold")
		case repository.ErrUserNotFound:
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "item purchased successfully")
}

// TakeOnEquipmentItem godoc
// @Summary Equip an item
// @Description Equip an equipment item from inventory
// @Tags equipment
// @Accept json
// @Produce json
// @Security Bearer
// @Param slug path string true "Item slug"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/equipment_items/{slug}/take_on [post]
func (h *EquipmentItemHandler) TakeOnEquipmentItem(c echo.Context) error {
	itemSlug := c.Param("slug")
	if itemSlug == "" {
		return ErrBadRequest(c, "item slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	inFight, fightErr := h.userRepo.InFight(userID)
	if fightErr != nil {
		return ErrInternalServerError(c)
	}
	if inFight {
		return ErrBadRequest(c, "user is in fight")
	}

	equipmentItemRepo := repository.NewEquipmentItemRepository(h.db)
	item, err := equipmentItemRepo.FindBySlug(itemSlug)
	if err != nil {
		return ErrNotFound(c, "equipment item not found")
	}

	err = h.equipmentItemTakeOnService.TakeOnEquipmentItem(c.Request().Context(), userID, item.ID)
	if err != nil {
		switch err {
		case services.ErrEquipmentItemNotFound:
			return ErrNotFound(c, "equipment item not found")
		case services.ErrItemNotInInventory:
			return ErrBadRequest(c, "item not in inventory")
		case services.ErrInsufficientLevel:
			return ErrBadRequest(c, "insufficient level")
		case services.ErrInvalidEquipmentType:
			return ErrBadRequest(c, "invalid equipment type")
		case repository.ErrUserNotFound:
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "item equipped successfully")
}

// TakeOffEquipmentItem godoc
// @Summary Unequip an item
// @Description Remove an equipment item from equipped slot
// @Tags equipment
// @Accept json
// @Produce json
// @Security Bearer
// @Param slot path string true "Equipment slot"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/equipment_items/take_off/{slot} [post]
func (h *EquipmentItemHandler) TakeOffEquipmentItem(c echo.Context) error {
	slotName := c.Param("slot")
	if slotName == "" {
		return ErrBadRequest(c, "slot name is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	inFight, fightErr := h.userRepo.InFight(userID)
	if fightErr != nil {
		return ErrInternalServerError(c)
	}
	if inFight {
		return ErrBadRequest(c, "user is in fight")
	}

	err = h.equipmentItemTakeOffService.TakeOffEquipmentItem(c.Request().Context(), userID, slotName)
	if err != nil {
		switch err {
		case services.ErrNoItemEquipped:
			return ErrBadRequest(c, "no item equipped in this slot")
		case services.ErrInvalidEquipmentType:
			return ErrBadRequest(c, "invalid slot name")
		case repository.ErrUserNotFound:
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "item removed successfully")
}

// SellEquipmentItem godoc
// @Summary Sell equipment item
// @Description Sell an equipment item from inventory
// @Tags equipment
// @Accept json
// @Produce json
// @Security Bearer
// @Param slug path string true "Item slug"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/equipment_items/{slug}/sell [post]
func (h *EquipmentItemHandler) SellEquipmentItem(c echo.Context) error {
	itemSlug := c.Param("slug")
	if itemSlug == "" {
		return ErrBadRequest(c, "item slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	inFight, fightErr := h.userRepo.InFight(userID)
	if fightErr != nil {
		return ErrInternalServerError(c)
	}
	if inFight {
		return ErrBadRequest(c, "user is in fight")
	}

	err = h.equipmentItemSellService.SellEquipmentItem(c.Request().Context(), userID, itemSlug)
	if err != nil {
		switch err {
		case services.ErrItemNotOwned:
			return ErrBadRequest(c, "item not owned")
		case services.ErrEquipmentItemNotFound:
			return ErrNotFound(c, "equipment item not found")
		case repository.ErrUserNotFound:
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "item sold successfully")
}
