package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/IBM/sarama"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jcmturner/gokrb5/v8/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Set time.Local to time.UTC
	time.Local = time.UTC

	// Load configuration
	conf := config.New()

	// Dial the gRPC server
	grpcConn, err := grpc.Dial(
		conf.GetString("config-service.host")+":"+conf.GetString("config-service.port"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to Config Service gRPC server: %v", err)
	}
	log.Println("Connected to Config Service gRPC server")

	// Create a gRPC client
	client := social_media_proto.NewConfigClient(grpcConn)
	// Create metadata

	// Add metadata to the context
	ctx := createMetadataContext(conf)

	// Call the Get method on the server
	response, err := client.Get(ctx, &empty.Empty{})
	if err != nil {
		log.Fatalf("Error calling Get: %v", err)
	}
	grpcConn.Close()

	// Parse the response
	extConf, err := parseConfigResponse(response)
	if err != nil {
		log.Fatalf("Error unmarshaling configuration: %v", err)
	}

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
