package main

import (
	k "KafkaExample/internal/kafka"
	handler2 "KafkaExample/internal/kafkaHandler"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var address = []string{"localhost:9091", "localhost:9092", "localhost:9093"}

const (
	consumerGroup = "KafkaExample"
	topic         = "my-topic"
)

func main() {
	handler := handler2.NewHandler()
	c1, err := k.NewConsumer(address, handler, consumerGroup, topic, 1)
	if err != nil {
		logrus.Fatal(err)
	}
	c2, err := k.NewConsumer(address, handler, consumerGroup, topic, 2)
	if err != nil {
		logrus.Fatal(err)
	}
	c3, err := k.NewConsumer(address, handler, consumerGroup, topic, 3)
	if err != nil {
		logrus.Fatal(err)
	}

	go func() {
		c1.Start()
	}()
	go func() {
		c2.Start()
	}()
	go func() {
		c3.Start()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan
	logrus.Info("Shutting down...")
	wg := &sync.WaitGroup{}
	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func() {
			defer wg.Done()
			err := c1.Stop()
			logrus.Infof("Stopping consumer %d: %v", i, err)
		}()
	}
	wg.Wait()

}
