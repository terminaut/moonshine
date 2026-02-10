package handlers

import (
	"context"
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
	db                          *sqlx.DB
	equipmentItemService        *services.EquipmentItemService
	equipmentItemBuyService     *services.EquipmentItemBuyService
	equipmentItemSellService    *services.EquipmentItemSellService
	equipmentItemTakeOnService  *services.EquipmentItemTakeOnService
	equipmentItemTakeOffService *services.EquipmentItemTakeOffService
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
		db:                          db,
		equipmentItemService:        equipmentItemService,
		equipmentItemBuyService:     equipmentItemBuyService,
		equipmentItemSellService:    equipmentItemSellService,
		equipmentItemTakeOnService:  equipmentItemTakeOnService,
		equipmentItemTakeOffService: equipmentItemTakeOffService,
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
