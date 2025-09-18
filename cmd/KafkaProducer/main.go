package main

import (
	"encoding/json"
	"fmt"
	"os"
	"wb_l0/configs"
	"wb_l0/configs/loader/dotEnvLoader"
	k "wb_l0/internal/delivery/kafka"
	"wb_l0/internal/domain"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func main() {

	envLoader := dotEnvLoader.DotEnvLoader{}
	cfg := configs.MustLoad(envLoader)
	cfg.KF.BootstrapServers = "localhost:9091, localhost:9092,localhost:9093"

	p, err := k.NewProducer(cfg)
	if err != nil {
		logrus.Fatal(err)
	}
	numberOfKeys := cfg.KF.ProducerNumberOfKeys
	uuids := generateKeys(numberOfKeys)
	for i := 1; i < 150; i++ {
		order := domain.CreateTestOrder(i)
		orderString, err := json.Marshal(order)
		if err != nil {
			fmt.Printf("error marshalling order %v: %v\n", order, err)
			os.Exit(1)
		}
		key := uuids[i%len(uuids)]
		if err = p.Produce(string(orderString), cfg.KF.Topic, key); err != nil {
			fmt.Printf("Error producing order %v: %v\n", order, err)
		}
		fmt.Printf("Producing order %v to Kafka\n", order)
	}
}

func generateKeys(numberOfKeys int) []string {
	keys := make([]string, numberOfKeys)
	for i := 0; i < numberOfKeys; i++ {
		keys[i] = uuid.NewString()
	}
	return keys
}
