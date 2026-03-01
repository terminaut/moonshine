package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

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
	userCache                   r.Cache[domain.User]
	fightChecker                *FightChecker
}

func NewEquipmentItemHandler(
	equipmentItemService *services.EquipmentItemService,
	equipmentItemBuyService *services.EquipmentItemBuyService,
	equipmentItemSellService *services.EquipmentItemSellService,
	equipmentItemTakeOnService *services.EquipmentItemTakeOnService,
	equipmentItemTakeOffService *services.EquipmentItemTakeOffService,
	equipmentItemRepo *repository.EquipmentItemRepository,
	userCache r.Cache[domain.User],
	fightChecker *FightChecker,
) *EquipmentItemHandler {
	return &EquipmentItemHandler{
		equipmentItemService:        equipmentItemService,
		equipmentItemBuyService:     equipmentItemBuyService,
		equipmentItemSellService:    equipmentItemSellService,
		equipmentItemTakeOnService:  equipmentItemTakeOnService,
		equipmentItemTakeOffService: equipmentItemTakeOffService,
		equipmentItemRepo:           equipmentItemRepo,
		userCache:                   userCache,
		fightChecker:                fightChecker,
	}
}

func (h *EquipmentItemHandler) invalidateUserCache(ctx context.Context, userID string) {
	if h.userCache != nil {
		_ = h.userCache.Delete(ctx, userID)
	}
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

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
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

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	err = h.equipmentItemBuyService.BuyEquipmentItem(c.Request().Context(), userID, itemSlug)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEquipmentItemNotFound):
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

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	item, err := h.equipmentItemRepo.FindBySlug(itemSlug)
	if err != nil {
		return ErrNotFound(c, "equipment item not found")
	}

	err = h.equipmentItemTakeOnService.TakeOnEquipmentItem(c.Request().Context(), userID, item.ID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEquipmentItemNotFound):
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

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
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

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	err = h.equipmentItemSellService.SellEquipmentItem(c.Request().Context(), userID, itemSlug)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrItemNotOwned):
			return ErrBadRequest(c, "item not owned")
		case errors.Is(err, repository.ErrEquipmentItemNotFound):
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
