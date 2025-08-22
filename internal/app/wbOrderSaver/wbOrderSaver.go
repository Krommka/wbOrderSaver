package producer

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
	"wb_l0/internal/domain"
	"wb_l0/internal/repository/postgres"
	"wb_l0/pkg/logger"
)

func Run() {

	loader := dotEnvLoader.DotEnvLoader{}
	cfg := configs.MustLoad(loader)
	log := logger.NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := postgres.NewStore(ctx, cfg)
	if err != nil {
		log.Error("failed to connect to database")
		os.Exit(1)
	}

	//cache, err := redisCache.NewCache(ctx, cfg, "movie:", log)
	//if err == nil {
	//	cachedRepo := cachedRepo.NewCachedRepo(db, cache, log)
	//	actor = usecase.NewActor(cachedRepo)
	//	film = usecase.NewFilm(cachedRepo)
	//} else {
	//	actor = usecase.NewActor(repo)
	//	film = usecase.NewFilm(repo)
	//}

	//httpSrv := &http.Server{
	//	Addr:    ":8081",
	//	Handler: http.Handler(nil),
	//}
	//
	//go func() {
	//	log.Info("Запуск сервера на порту 8081")
	//	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
	//		log.Error("HTTP server error: ", err)
	//		os.Exit(1)
	//	}
	//}()

	testOrder := createTestOrder()

	// Вставляем заказ в базу
	if err := db.CreateOrder(ctx, testOrder); err != nil {
		log.Error("failed to create order:", err)
	}

	log.Info("Order created successfully!")

	var order *domain.Order

	if order, err = db.GetOrder(ctx, "b563feb7b2b84b6b563a"); err != nil {
		log.Error("failed to get order:", err)
	}
	fmt.Println(order)
	//
	if err = db.DeleteOrder(ctx, "b563feb7b2b84b6b563a"); err != nil {
		log.Error("failed to delete order:", err)
	}
	//
	if order, err = db.GetOrder(ctx, "b563feb7b2b84b6b563a"); err != nil {
		log.Error("failed to get order:", err)
	}
	//

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("Остановка сервисов")

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Disconnect(ctx)
	}()

	completed := make(chan struct{})

	go func() {
		wg.Wait()
		close(completed)
	}()

	select {
	case <-completed:
		log.Info("Все сервисы корректно остановлены")
	case <-shutdownCtx.Done():
		log.Info("Таймаут заверешения работы превышен, принудительная остановка")
	}

}

func createTestOrder() domain.Order {
	now := time.Now().UTC()
	return domain.Order{
		OrderUID:          "b563feb7b2b84b6b563a",
		TrackNumber:       "WBILMTESTTRACK",
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		ShardKey:          "9",
		SMID:              99,
		DateCreated:       now,
		OOFShard:          "1",
		Delivery: domain.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: domain.Payment{
			Transaction:  "b563feb7b2b84b6b563a",
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDT:    now.Unix(),
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []domain.Item{
			{
				ChrtID:      9934930,
				TrackNumber: "WBILMTESTTRACK",
				Price:       453,
				RID:         "ab4219087a764ae0b473",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NMID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
	}
}
