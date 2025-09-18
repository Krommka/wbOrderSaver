package kafka

import (
	"fmt"
	"wb_l0/configs"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
)

const (
	noTimeout = -1
)

type Handler interface {
	HandleMessage(message []byte, topic kafka.TopicPartition, cn int) error
}

type Consumer struct {
	consumer       *kafka.Consumer
	handler        Handler
	stop           bool
	consumerNumber int
}

func NewConsumer(cfg *configs.Config, handler Handler, consumerNumber int) (*Consumer,
	error) {

	config := &kafka.ConfigMap{
		"bootstrap.servers":        cfg.KF.BootstrapServers,
		"group.id":                 cfg.KF.ConsumerGroup,
		"session.timeout.ms":       cfg.KF.SessionTimeoutMs,
		"enable.auto.offset.store": false,
		"enable.auto.commit":       true,
		"auto.commit.interval.ms":  cfg.KF.AutoCommitIntervalMs,
		"auto.offset.reset":        cfg.KF.AutoOffsetReset,
	}

	c, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, fmt.Errorf("error creating consumer: %v", err)
	}
	if err = c.Subscribe(cfg.KF.Topic, nil); err != nil {
		return nil, fmt.Errorf("error subscribing to topic: %v", err)
	}
	return &Consumer{
		consumer:       c,
		handler:        handler,
		consumerNumber: consumerNumber,
	}, nil
}

func (c *Consumer) Start() {
	for {
		if c.stop {
			break
		}
		kafkaMsg, err := c.consumer.ReadMessage(noTimeout)
		if err != nil {
			logrus.Errorf("error reading message from kafka %v", err)
		}
		if kafkaMsg == nil {
			continue
		}
		if err := c.handler.HandleMessage(kafkaMsg.Value, kafkaMsg.TopicPartition, c.consumerNumber); err != nil {
			logrus.Errorf("error reading message from kafka %v", err)
			continue
		}
		if _, err = c.consumer.StoreMessage(kafkaMsg); err != nil {
			logrus.Errorf("error storing message to kafka %v", err)
			continue
		}
	}
}

func (c *Consumer) Stop() error {
	c.stop = true
	if _, err := c.consumer.Commit(); err != nil {
		return err
	}
	logrus.Info("Commited offset")
	return c.consumer.Close()
}
