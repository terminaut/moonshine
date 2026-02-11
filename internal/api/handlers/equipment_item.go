package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/domain"
	r "moonshine/internal/redis"
	"moonshine/internal/repository"
)

type EquipmentItemHandler struct {
	equipmentItemService        *services.EquipmentItemService
	equipmentItemBuyService     *services.EquipmentItemBuyService
	equipmentItemSellService    *services.EquipmentItemSellService
	equipmentItemTakeOnService  *services.EquipmentItemTakeOnService
	equipmentItemTakeOffService *services.EquipmentItemTakeOffService
	equipmentItemRepo           *repository.EquipmentItemRepository
	userRepo                    *repository.UserRepository
	userCache                   r.Cache[domain.User]
}

func NewEquipmentItemHandler(db *sqlx.DB, rdb *redis.Client) *EquipmentItemHandler {
	equipmentItemRepo := repository.NewEquipmentItemRepository(db)
	equipmentItemService := services.NewEquipmentItemService(equipmentItemRepo)

	inventoryRepo := repository.NewInventoryRepository(db)
	userRepo := repository.NewUserRepository(db)
	equipmentItemBuyService := services.NewEquipmentItemBuyService(db, equipmentItemRepo, inventoryRepo, userRepo)
	equipmentItemSellService := services.NewEquipmentItemSellService(db, equipmentItemRepo, inventoryRepo, userRepo)
	equipmentItemTakeOnService := services.NewEquipmentItemTakeOnService(db, equipmentItemRepo, inventoryRepo, userRepo)
	equipmentItemTakeOffService := services.NewEquipmentItemTakeOffService(db, equipmentItemRepo, inventoryRepo, userRepo)

	return &EquipmentItemHandler{
		equipmentItemService:        equipmentItemService,
		equipmentItemBuyService:     equipmentItemBuyService,
		equipmentItemSellService:    equipmentItemSellService,
		equipmentItemTakeOnService:  equipmentItemTakeOnService,
		equipmentItemTakeOffService: equipmentItemTakeOffService,
		equipmentItemRepo:           equipmentItemRepo,
		userRepo:                    userRepo,
		userCache:                   r.NewJSONCache[domain.User](rdb, "user", 5*time.Second),
	}
}

func (h *EquipmentItemHandler) invalidateUserCache(ctx context.Context, userID string) {
	_ = h.userCache.Delete(ctx, userID)
}

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
		switch {
		case errors.Is(err, services.ErrEquipmentItemNotFound):
			return ErrNotFound(c, "equipment item not found")
		case errors.Is(err, services.ErrInsufficientGold):
			return ErrBadRequest(c, "insufficient gold")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "item purchased successfully")
}

func (h *EquipmentItemHandler) TakeOnEquipmentItem(c echo.Context) error {
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

	item, err := h.equipmentItemRepo.FindBySlug(itemSlug)
	if err != nil {
		return ErrNotFound(c, "equipment item not found")
	}

	err = h.equipmentItemTakeOnService.TakeOnEquipmentItem(c.Request().Context(), userID, item.ID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrEquipmentItemNotFound):
			return ErrNotFound(c, "equipment item not found")
		case errors.Is(err, services.ErrItemNotInInventory):
			return ErrBadRequest(c, "item not in inventory")
		case errors.Is(err, services.ErrInsufficientLevel):
			return ErrBadRequest(c, "insufficient level")
		case errors.Is(err, services.ErrInvalidEquipmentType):
			return ErrBadRequest(c, "invalid equipment type")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "item equipped successfully")
}

func (h *EquipmentItemHandler) TakeOffEquipmentItem(c echo.Context) error {
	slotName := c.Param("slot")
	if slotName == "" {
		return ErrBadRequest(c, "slot name is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
		return err
	}

	err = h.equipmentItemTakeOffService.TakeOffEquipmentItem(c.Request().Context(), userID, slotName)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNoItemEquipped):
			return ErrBadRequest(c, "no item equipped in this slot")
		case errors.Is(err, services.ErrInvalidEquipmentType):
			return ErrBadRequest(c, "invalid slot name")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "item removed successfully")
}

func (h *EquipmentItemHandler) SellEquipmentItem(c echo.Context) error {
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

	err = h.equipmentItemSellService.SellEquipmentItem(c.Request().Context(), userID, itemSlug)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrItemNotOwned):
			return ErrBadRequest(c, "item not owned")
		case errors.Is(err, services.ErrEquipmentItemNotFound):
			return ErrNotFound(c, "equipment item not found")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "item sold successfully")
}
