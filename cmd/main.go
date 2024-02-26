package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/IBM/sarama"
	"github.com/PNYwise/post-consumer/internal/config"
	"github.com/PNYwise/post-consumer/internal/domain"
	post_consumer "github.com/PNYwise/post-consumer/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
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
	client := post_consumer.NewConfigClient(grpcConn)
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
	brokerList := []string{
		fmt.Sprintf("%s:%d", extConf.Kafka.Host, extConf.Kafka.Port),
	}

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
	offset := int64(sarama.OffsetNewest)

	// Create a partition consumer for the given topic, partition, and offset
	partitionConsumer, err := consumer.ConsumePartition(extConf.Kafka.Topic[0], extConf.Kafka.Partition, offset)
	if err != nil {
		log.Fatalf("Error creating partition consumer: %v", err)
	}
	partitionConsumer2, err := consumer.ConsumePartition(extConf.Kafka.Topic[1], extConf.Kafka.Partition, offset)
	if err != nil {
		log.Fatalf("Error creating partition consumer: %v", err)
	}
	defer func() {
		if err := partitionConsumer.Close(); err != nil {
			log.Fatalf("Error closing partition consumer: %v", err)
		}
		if err := partitionConsumer2.Close(); err != nil {
			log.Fatalf("Error closing partition 2 consumer: %v", err)
		}
	}()

	log.Println("Post Consumer Run...")

	// Wait for messages and print them
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

ConsumerLoop:
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			log.Printf("Received message: %s", msg.Value)
			// TODO: save data into database
		case msg := <-partitionConsumer2.Messages():
			log.Printf("Received message: %s", msg.Value)
			// TODO: save data into database
		case err := <-partitionConsumer.Errors():
			log.Printf("Error consuming message: %v", err)
		case err := <-partitionConsumer2.Errors():
			log.Printf("Error consuming message: %v", err)
		case <-signals:
			break ConsumerLoop
		}
	}
}

func createMetadataContext(conf *viper.Viper) context.Context {
	// Add metadata to the context
	return metadata.NewOutgoingContext(context.Background(), metadata.New(map[string]string{
		"id":    conf.GetString("id"),
		"token": conf.GetString("token"),
	}))
}

func parseConfigResponse(response *structpb.Value) (*domain.ExtConf, error) {
	extConf := &domain.ExtConf{}
	if stringVal, ok := response.Kind.(*structpb.Value_StringValue); ok {
		err := json.Unmarshal([]byte(stringVal.StringValue), extConf)
		return extConf, err
	}
	return nil, nil
}
