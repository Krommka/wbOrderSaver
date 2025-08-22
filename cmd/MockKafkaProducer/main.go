package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"os"
	"time"
	"wb_l0/configs"
	"wb_l0/configs/loader/dotEnvLoader"
	k "wb_l0/internal/delivery/kafka"
	"wb_l0/internal/domain"
	"wb_l0/pkg/logger"
)

func main() {

	envLoader := dotEnvLoader.DotEnvLoader{}
	cfg := configs.MustLoad(envLoader)
	log := logger.NewLogger(cfg)

	p, err := k.NewProducer(cfg)
	if err != nil {
		logrus.Fatal(err)
	}
	numberOfKeys := cfg.KF.ProducerNumberOfKeys
	uuids := generateUUID(numberOfKeys)
	order := createTestOrder()
	orderString, err := json.Marshal(order)
	if err != nil {
		log.Error("Error marshalling order", "error", err, "order", order)
		os.Exit(1)
	}
	key := uuids[0]
	if err = p.Produce(string(orderString), cfg.KF.Topic, key); err != nil {
		log.Error("Error producing order", "error", err, "order", order)
	}

}

func generateUUID(numberOfKeys int) []string {
	uuids := make([]string, numberOfKeys)
	for i := 0; i < numberOfKeys; i++ {
		uuids[i] = uuid.NewString()
	}
	return uuids
}

func createTestOrder() domain.Order {
	now := time.Now().UTC()
	return domain.Order{
		OrderUID:          "b563feb7b2b84b6b563b",
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
