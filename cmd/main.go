package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/IBM/sarama"
)

func main() {
	// Kafka broker address
	brokerList := []string{"127.0.0.1:9092"}

	// Create a new Kafka consumer
	config := sarama.NewConfig()
	consumer, err := sarama.NewConsumer(brokerList, config)
	if err != nil {
		log.Fatalf("Error creating Kafka consumer: %v", err)
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Fatalf("Error closing Kafka consumer: %v", err)
		}
	}()

	// Kafka topic to consume messages from
	topic := "post"
	partition := int32(0)
	offset := int64(sarama.OffsetNewest)

	// Create a partition consumer for the given topic, partition, and offset
	partitionConsumer, err := consumer.ConsumePartition(topic, partition, offset)
	if err != nil {
		log.Fatalf("Error creating partition consumer: %v", err)
	}
	defer func() {
		if err := partitionConsumer.Close(); err != nil {
			log.Fatalf("Error closing partition consumer: %v", err)
		}
	}()

	// Wait for messages and print them
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

ConsumerLoop:
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			log.Printf("Received message: %s", msg.Value)
			// TODO: save data into database
		case err := <-partitionConsumer.Errors():
			log.Printf("Error consuming message: %v", err)
		case <-signals:
			break ConsumerLoop
		}
	}
}
