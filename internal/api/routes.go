package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"

	"moonshine/internal/api/handlers"
	jwtMiddleware "moonshine/internal/api/middleware"
)

func SetupRoutes(e *echo.Echo, c *Container) {
	e.GET("/health", healthCheck)

	wsHandler := handlers.NewWebSocketHandler(c.Hub, c.Config)
	e.GET("/api/ws", wsHandler.HandleConnection)

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			if strings.HasPrefix(ctx.Request().URL.Path, "/assets") {
				if c.Config.IsProduction() {
					ctx.Response().Header().Set("Cache-Control", "public, max-age=604800")
				} else {
					ctx.Response().Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
					ctx.Response().Header().Set("Pragma", "no-cache")
					ctx.Response().Header().Set("Expires", "0")
				}
			}
			return next(ctx)
		}
	})

	var assetsPath string
	possiblePaths := []string{
		"frontend/assets",
		"../frontend/assets",
		filepath.Join(filepath.Dir(os.Args[0]), "../frontend/assets"),
	}

	for _, path := range possiblePaths {
		absPath, err := filepath.Abs(path)
		if err == nil {
			if _, err := os.Stat(filepath.Join(absPath, "images")); err == nil {
				assetsPath = absPath
				break
			}
		}
	}

	if assetsPath == "" {
		wd, _ := os.Getwd()
		for {
			if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
				assetsPath, _ = filepath.Abs(filepath.Join(wd, "frontend/assets"))
				if _, err := os.Stat(assetsPath); err == nil {
					break
				}
			}
			parent := filepath.Dir(wd)
			if parent == wd {
				break
			}
			wd = parent
		}
	}

	if assetsPath != "" {
		e.Static("/assets", assetsPath)
	} else {
		e.Static("/assets", "frontend/assets")
	}

	e.Validator = NewValidator()

	fightChecker := handlers.NewFightChecker(c.UserRepo)

	authHandler := handlers.NewAuthHandler(c.AuthService, c.LocationRepo, c.UserRepo)
	authGroup := e.Group("/api/auth")
	authGroup.POST("/signup", authHandler.SignUp)
	authGroup.POST("/signin", authHandler.SignIn)

	jwtConfig := echojwt.Config{
		SigningKey: []byte(c.Config.JWTKey),
		ContextKey: "user",
		ErrorHandler: func(ctx echo.Context, err error) error {
			return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		},
	}

	apiGroup := e.Group("/api")
	apiGroup.Use(echojwt.WithConfig(jwtConfig))
	apiGroup.Use(jwtMiddleware.ExtractUserIDFromJWT())

	userHandler := handlers.NewUserHandler(c.UserService, c.InventoryService, c.UserRepo, c.EquipmentItemRepo, c.PotionRepo, c.ToolItemRepo, fightChecker)
	apiGroup.GET("/user/me", userHandler.GetCurrentUser)
	apiGroup.PUT("/user/me", userHandler.UpdateCurrentUser)
	apiGroup.GET("/users/me/inventory", userHandler.GetUserInventory)
	apiGroup.GET("/users/me/equipped", userHandler.GetUserEquippedItems)

	avatarHandler := handlers.NewAvatarHandler(c.AvatarService, fightChecker)
	apiGroup.GET("/avatars", avatarHandler.GetAllAvatars)

	locationHandler := handlers.NewLocationHandler(c.LocationService, c.LocationRepo, c.UserRepo, fightChecker)
	apiGroup.POST("/locations/:slug/move", locationHandler.MoveToLocation)
	apiGroup.POST("/locations/:slug/cells/:cell_slug/move", locationHandler.MoveToCell)
	apiGroup.GET("/locations/:slug/cells", locationHandler.GetLocationCells)

	equipmentItemHandler := handlers.NewEquipmentItemHandler(
		c.EquipmentItemService,
		c.EquipmentItemBuyService,
		c.EquipmentItemSellService,
		c.EquipmentItemTakeOnService,
		c.EquipmentItemTakeOffService,
		c.EquipmentItemRepo,
		c.UserCache,
		fightChecker,
	)
	apiGroup.GET("/equipment_items", equipmentItemHandler.GetEquipmentItems)
	apiGroup.POST("/equipment_items/take_off/:slot", equipmentItemHandler.TakeOffEquipmentItem)
	apiGroup.POST("/equipment_items/:slug/buy", equipmentItemHandler.BuyEquipmentItem)
	apiGroup.POST("/equipment_items/:slug/sell", equipmentItemHandler.SellEquipmentItem)
	apiGroup.POST("/equipment_items/:slug/take_on", equipmentItemHandler.TakeOnEquipmentItem)

	potionHandler := handlers.NewPotionHandler(
		c.PotionBuyService,
		c.PotionSellService,
		c.PotionTakeOnService,
		c.PotionTakeOffService,
		c.PotionUseService,
		c.PotionRepo,
		c.UserCache,
		fightChecker,
	)
	apiGroup.GET("/potions", potionHandler.GetPotions)
	apiGroup.POST("/potions/take_off/:slot", potionHandler.TakeOffPotion)
	apiGroup.POST("/potions/use/:slot", potionHandler.UsePotion)
	apiGroup.POST("/potions/:slug/buy", potionHandler.BuyPotion)
	apiGroup.POST("/potions/:slug/sell", potionHandler.SellPotion)
	apiGroup.POST("/potions/:slug/take_on", potionHandler.TakeOnPotion)

	toolItemHandler := handlers.NewToolItemHandler(
		c.ToolItemBuyService,
		c.ToolItemSellService,
		c.ToolItemTakeOnService,
		c.ToolItemTakeOffService,
		c.ToolItemRepo,
		c.UserCache,
		fightChecker,
	)
	apiGroup.GET("/tool_items", toolItemHandler.GetToolItems)
	apiGroup.POST("/tool_items/:slug/buy", toolItemHandler.BuyToolItem)
	apiGroup.POST("/tool_items/:slug/sell", toolItemHandler.SellToolItem)
	apiGroup.POST("/tool_items/:slug/take_on", toolItemHandler.TakeOnToolItem)
	apiGroup.POST("/tool_items/take_off", toolItemHandler.TakeOffToolItem)

	resourceHandler := handlers.NewResourceHandler(c.ResourceService, fightChecker)
	apiGroup.GET("/resources/:location_slug", resourceHandler.GetResources)
	apiGroup.POST("/resources/:slug/gather", resourceHandler.Gather)

	botHandler := handlers.NewBotHandler(c.BotService, fightChecker)
	apiGroup.GET("/bots/:location_slug", botHandler.GetBots)
	apiGroup.POST("/bots/:slug/attack", botHandler.Attack)

	fightHandler := handlers.NewFightHandler(c.FightService, c.LocationRepo)
	apiGroup.GET("/fights/current", fightHandler.GetCurrentFight)
	apiGroup.POST("/fights/current/hit", fightHandler.Hit)
}

func healthCheck(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}
