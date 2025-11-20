package wbOrderSaver

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"wb_l0/configs"
	"wb_l0/configs/loader/dotEnvLoader"
	h "wb_l0/internal/delivery/http"
	k "wb_l0/internal/delivery/kafka"
	"wb_l0/internal/delivery/kafka/kafkaHandler"
	"wb_l0/internal/repository/cachedRepo"
	"wb_l0/internal/repository/postgres"
	"wb_l0/internal/repository/redisCache"
	"wb_l0/internal/usecase"
	"wb_l0/pkg/logger"
)

func Run() {

	envLoader := dotEnvLoader.DotEnvLoader{}
	cfg := configs.MustLoad(envLoader)
	log := logger.NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := postgres.NewStore(ctx, cfg, log)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	cache, err := redisCache.NewCache(ctx, cfg, "order:", log)
	var orderUsecase *usecase.OrderUsecase
	if err == nil {
		repo := cachedRepo.NewCachedRepo(ctx, db, cache, log, cfg)
		orderUsecase = usecase.NewOrderUsecase(repo, 3, log)

	} else {
		orderUsecase = usecase.NewOrderUsecase(db, 3, log)
	}

	handler := kafkaHandler.NewKafkaHandler(orderUsecase, log)
	c1, err := k.NewConsumer(cfg, handler, 1)
	if err != nil {
		log.Error("failed to connect to consumer")
		os.Exit(1)
	}

	go func() {
		c1.Start()
	}()

	router := h.SetupRouter(orderUsecase, log)

	server := &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	go func() {
		if serverErr := server.ListenAndServe(); serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			log.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
		log.Info("Server started", "port", cfg.HTTP.Port)
	}()

	//prometheus.InitPrometheus()
	httpSrv := &http.Server{
		Addr:    ":8082",
		Handler: promhttp.Handler(),
	}

	go func() {
		log.Info("Запуск prometheus", "port", 8082)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP prometheus server error: ", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("Stopping services")

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()
	wg := &sync.WaitGroup{}

	wg.Add(4)
	go func() {
		defer wg.Done()
		db.Disconnect(ctx)
	}()

	go func() {
		defer wg.Done()
		if consumerErr := c1.Stop(); err != nil {
			log.Error("failed to stop consumer", "error", consumerErr)
		}
	}()

	go func() {
		defer wg.Done()
		log.Info("Shutting down server...")

		if serverErr := server.Shutdown(ctx); serverErr != nil {
			log.Error("Server shutdown error", "error", serverErr)
		}

		log.Info("Server stopped")
	}()

	go func() {
		defer wg.Done()
		log.Info("Shutting down prometheus server...")
		if serverErr := httpSrv.Shutdown(ctx); serverErr != nil {
			log.Error("Server shutdown error", "error", serverErr)
		}
		log.Info("Prometheus server stopped")
	}()

	completed := make(chan struct{})

	go func() {
		wg.Wait()
		close(completed)
	}()

	select {
	case <-completed:
		log.Info("All services correctly stopped")
	case <-shutdownCtx.Done():
		log.Info("Shutdown timeout exceeded, forced stop")
	}

}
