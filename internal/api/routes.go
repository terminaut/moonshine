package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"moonshine/internal/api/handlers"
	jwtMiddleware "moonshine/internal/api/middleware"
	"moonshine/internal/config"
)

func SetupRoutes(e *echo.Echo, db *sqlx.DB, rdb *redis.Client, cfg *config.Config) {
	e.GET("/health", healthCheck)

	wsHandler := handlers.NewWebSocketHandler(cfg)
	e.GET("/api/ws", wsHandler.HandleConnection)

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if strings.HasPrefix(c.Request().URL.Path, "/assets") {
				if cfg.IsProduction() {
					c.Response().Header().Set("Cache-Control", "public, max-age=604800")
				} else {
					c.Response().Header().Set("Cache-Control", "public, max-age=3600")
				}
			}
			return next(c)
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

	authHandler := handlers.NewAuthHandler(db, cfg.JWTKey)
	authGroup := e.Group("/api/auth")
	authGroup.POST("/signup", authHandler.SignUp)
	authGroup.POST("/signin", authHandler.SignIn)

	jwtConfig := echojwt.Config{
		SigningKey: []byte(cfg.JWTKey),
		ContextKey: "user",
		ErrorHandler: func(c echo.Context, err error) error {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		},
	}

	apiGroup := e.Group("/api")
	apiGroup.Use(echojwt.WithConfig(jwtConfig))
	apiGroup.Use(jwtMiddleware.ExtractUserIDFromJWT())

	userHandler := handlers.NewUserHandler(db, rdb)
	apiGroup.GET("/user/me", userHandler.GetCurrentUser)
	apiGroup.PUT("/user/me", userHandler.UpdateCurrentUser)
	apiGroup.GET("/users/me/inventory", userHandler.GetUserInventory)
	apiGroup.GET("/users/me/equipped", userHandler.GetUserEquippedItems)

	avatarHandler := handlers.NewAvatarHandler(db)
	apiGroup.GET("/avatars", avatarHandler.GetAllAvatars)

	locationHandler := handlers.NewLocationHandler(db, rdb)
	apiGroup.POST("/locations/:slug/move", locationHandler.MoveToLocation)
	apiGroup.POST("/locations/:slug/cells/:cell_slug/move", locationHandler.MoveToCell)
	apiGroup.GET("/locations/:slug/cells", locationHandler.GetLocationCells)

	equipmentItemHandler := handlers.NewEquipmentItemHandler(db, rdb)
	apiGroup.GET("/equipment_items", equipmentItemHandler.GetEquipmentItems)
	apiGroup.POST("/equipment_items/take_off/:slot", equipmentItemHandler.TakeOffEquipmentItem)
	apiGroup.POST("/equipment_items/:slug/buy", equipmentItemHandler.BuyEquipmentItem)
	apiGroup.POST("/equipment_items/:slug/sell", equipmentItemHandler.SellEquipmentItem)
	apiGroup.POST("/equipment_items/:slug/take_on", equipmentItemHandler.TakeOnEquipmentItem)

	botHandler := handlers.NewBotHandler(db)
	apiGroup.GET("/bots/:location_slug", botHandler.GetBots)
	apiGroup.POST("/bots/:slug/attack", botHandler.Attack)

	fightHandler := handlers.NewFightHandler(db)
	apiGroup.GET("/fights/current", fightHandler.GetCurrentFight)
	apiGroup.POST("/fights/current/hit", fightHandler.Hit)
}

func healthCheck(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}
