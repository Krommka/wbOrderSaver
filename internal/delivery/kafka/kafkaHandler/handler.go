package kafkaHandler

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
	"time"
	"wb_l0/internal/domain"
)

type KafkaHandler struct {
	store      *store.Store
	retryCount int
}

func NewKafkaHandler(store *store.Store, retryCount int) *KafkaHandler {
	return &KafkaHandler{
		store:      store,
		retryCount: retryCount,
	}
}

func (h *Handler) HandleMessage(message []byte, topic kafka.TopicPartition, cn int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	order, err := h.parseOrder(message)
	if err != nil {
		log.Printf("Failed to parse order from Kafka: %v", err)
		return nil // Коммитим чтобы не зацикливаться на плохих сообщениях
	}
	if err := order.Validate(); err != nil {
		log.Printf("Invalid order %s: %v", order.OrderUID, err)
		return nil // Коммитим невалидные сообщения
	}
	if err := h.createOrderWithRetry(ctx, order); err != nil {
		return fmt.Errorf("failed to create order %s after %d retries: %w",
			order.OrderUID, h.retryCount, err)
	}

	log.Printf("Successfully processed order %s from partition %d",
		order.OrderUID, topic.Partition)
	return nil
}

func (h *KafkaHandler) createOrderWithRetry(ctx context.Context, order domain.Order) error {
	var lastErr error

	for i := 0; i < h.retryCount; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
			err := h.store.CreateOrder(ctx, order)
			if err == nil {
				return nil // Успех
			}

			lastErr = err
			log.Error("Retry %d/%d for order %s failed: %v",
				i+1, h.retryCount, order.OrderUID, err)

			// Экспоненциальная backoff задержка
			delay := time.Duration(1<<uint(i)) * time.Second
			time.Sleep(delay)
		}
	}

	return lastErr
}
