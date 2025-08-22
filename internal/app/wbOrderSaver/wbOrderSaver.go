package wbOrderSaver

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"wb_l0/configs"
	"wb_l0/configs/loader/dotEnvLoader"
	k "wb_l0/internal/delivery/kafka"
	handler2 "wb_l0/internal/delivery/kafka/kafkaHandler"
	"wb_l0/internal/domain"
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
		log.Error("failed to connect to database")
		os.Exit(1)
	}

	cache, err := redisCache.NewCache(ctx, cfg, "order:", log)
	var orderUsecase *usecase.OrderUsecase
	if err == nil {
		repo := cachedRepo.NewCachedRepo(db, cache, log)
		orderUsecase = usecase.NewOrderUsecase(repo, 3, log)
	} else {
		orderUsecase = usecase.NewOrderUsecase(db, 3, log)
	}

	handler := handler2.NewKafkaHandler(orderUsecase, log)
	c1, err := k.NewConsumer(cfg, handler, 1)
	if err != nil {
		log.Error("failed to connect to consumer")
		os.Exit(1)
	}

	go func() {
		c1.Start()
	}()
	orderUID := "00000000000000000001"
	order, err := orderUsecase.GetOrder(ctx, orderUID)
	if err != nil {
		if err != domain.ErrRecordNotFound {
			log.Error("error getting order", "error", err, "orderUID", orderUID)
		}
		log.Error("not found", "error", err)
	} else {
		fmt.Println(order)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("Stopping services")

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()
	wg := &sync.WaitGroup{}

	wg.Add(2)
	go func() {
		defer wg.Done()
		db.Disconnect(ctx)
	}()

	go func() {
		defer wg.Done()
		if err = c1.Stop(); err != nil {
			log.Error("failed to stop consumer", err)
		}
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
