package main

import (
	"context"
	"log"
	"moonshine/internal/redis"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"moonshine/cmd/server/docs"
	"moonshine/internal/api"
	"moonshine/internal/config"
	"moonshine/internal/metrics"
	"moonshine/internal/repository"
	"moonshine/internal/worker"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Moonshine API
// @version 1.0
// @description Game API for Moonshine
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description JWT token. Example: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatalf("no .env file found, using system envs")
	}

	cfg := config.Load()

	db, err := repository.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	rdb := redis.New(cfg)
	if err := redis.Ping(ctx, rdb); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer rdb.Close()

	docs.SwaggerInfo.Host = cfg.HTTPAddr
	if os.Getenv("ENV") == "production" {
		docs.SwaggerInfo.Schemes = []string{"https"}
	} else {
		docs.SwaggerInfo.Schemes = []string{"http"}
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(metrics.PrometheusMiddleware())

	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	api.SetupRoutes(e, db.DB(), rdb, cfg)

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	hpWorker := worker.NewHpWorker(db.DB(), 3*time.Second)
	go hpWorker.StartWorker(ctx)

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
}
