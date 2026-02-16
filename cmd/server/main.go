package main

import (
	"context"
	"errors"
	"log"
	"moonshine/internal/redis"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	"moonshine/cmd/server/docs"
	"moonshine/internal/api"
	"moonshine/internal/config"
	"moonshine/internal/metrics"
	"moonshine/internal/repository"
	"moonshine/internal/tracing"
	"moonshine/internal/worker"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("no .env file found, using system envs")
	}

	cfg := config.Load()

	var tracerShutdown func()
	if cfg.TracingEnabled {
		tp, err := tracing.InitTracer(ctx, cfg.JaegerEndpoint)
		if err != nil {
			log.Fatalf("failed to initialize tracing: %v", err)
		}
		tracerShutdown = func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = tp.Shutdown(shutdownCtx)
		}
		log.Println("tracing enabled, exporting to", cfg.JaegerEndpoint)
	}

	db, err := repository.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	rdb := redis.New(cfg)
	if err := redis.Ping(ctx, rdb); err != nil {
		db.Close()
		log.Fatalf("failed to connect to redis: %v", err)
	}

	defer stop()
	defer db.Close()
	defer func() { _ = rdb.Close() }()
	if tracerShutdown != nil {
		defer tracerShutdown()
	}

	docs.SwaggerInfo.Host = cfg.HTTPAddr
	if os.Getenv("ENV") == "production" {
		docs.SwaggerInfo.Schemes = []string{"https"}
	} else {
		docs.SwaggerInfo.Schemes = []string{"http"}
	}

	e := echo.New()
	e.Use(otelecho.Middleware("moonshine"))
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(metrics.PrometheusMiddleware())

	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	if cfg.PprofEnabled {
		registerPprof(e)
	}

	api.SetupRoutes(e, db.DB(), rdb, cfg)

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	hpWorker := worker.NewHpWorker(db.DB(), rdb, 3*time.Second)
	go hpWorker.StartWorker(ctx)

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	var shutdownErr error
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown failed: %v", err)
		shutdownErr = err
	}
	cancel()

	if shutdownErr != nil {
		os.Exit(1) //nolint:gocritic
	}
}

func registerPprof(e *echo.Echo) {
	e.GET("/debug/pprof/", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	e.GET("/debug/pprof/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	e.GET("/debug/pprof/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	e.GET("/debug/pprof/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	e.GET("/debug/pprof/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))
	e.GET("/debug/pprof/allocs", echo.WrapHandler(pprof.Handler("allocs")))
	e.GET("/debug/pprof/block", echo.WrapHandler(pprof.Handler("block")))
	e.GET("/debug/pprof/goroutine", echo.WrapHandler(pprof.Handler("goroutine")))
	e.GET("/debug/pprof/heap", echo.WrapHandler(pprof.Handler("heap")))
	e.GET("/debug/pprof/mutex", echo.WrapHandler(pprof.Handler("mutex")))
	e.GET("/debug/pprof/threadcreate", echo.WrapHandler(pprof.Handler("threadcreate")))
}
