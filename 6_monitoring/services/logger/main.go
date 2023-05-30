package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"
)

func main() {
	brokers := []string{os.Getenv("KAFKA_HOST")} // Modify with your Kafka broker addresses

	// Configure Kafka reader for Topic 1
	topic1 := os.Getenv("KAFKA_TOPIC_1")
	topic1Reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   brokers,
		Topic:     topic1,
		Partition: 0, // Adjust the partition as needed
		MinBytes:  10e3,
		MaxBytes:  10e6,
	})

	// Configure Kafka reader for Topic 2
	topic2 := os.Getenv("KAFKA_TOPIC_2")
	topic2Reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   brokers,
		Topic:     topic2,
		Partition: 0, // Adjust the partition as needed
		MinBytes:  10e3,
		MaxBytes:  10e6,
	})

	// Initialize context and signal channel for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// Start a goroutine to handle OS signals and initiate graceful shutdown
	go func() {
		<-signalCh
		log.Println("Shutting down...")
		cancel()
	}()

	// Start a goroutine to read from Topic 1
	go func() {
		defer topic1Reader.Close()

		for {
			msg, err := topic1Reader.ReadMessage(ctx)
			if err != nil {
				log.Println("Error reading message from Topic 1:", err)
				continue
			}

			log.Printf("[Topic 1] Received message: %s\n", string(msg.Value))
		}
	}()

	// Start a goroutine to read from Topic 2
	go func() {
		defer topic2Reader.Close()

		for {
			msg, err := topic2Reader.ReadMessage(ctx)
			if err != nil {
				log.Println("Error reading message from Topic 2:", err)
				continue
			}

			log.Printf("[Topic 2] Received message: %s\n", string(msg.Value))
		}
	}()

	// Wait for the termination signal
	<-ctx.Done()
	log.Println("Service stopped.")
}
