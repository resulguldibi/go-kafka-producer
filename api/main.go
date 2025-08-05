package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
)

// Event represents the incoming event structure
type Event struct {
	EventTimestamp int64  `json:"eventtimestamp"`
	EventTime      string `json:"eventtime"`
	ID             string `json:"id"`
	Domain         string `json:"domain"`
	Subdomain      string `json:"subdomain"`
	Code           string `json:"code"`
	Version        string `json:"version"`
	BranchID       int    `json:"branchid"`
	ChannelID      int    `json:"channelid"`
	CustomerID     int    `json:"customerid"`
	UserID         int    `json:"userid"`
	Payload        string `json:"payload"`
}

// EventResponse represents the response structure
type EventResponse struct {
	SuccessEventIds []string `json:"successEventIds"`
	InvalidEventIds []string `json:"invalidEventIds"`
	FailedEventIds  []string `json:"failedEventIds"`
}

// KafkaProducer wraps the kafka writer
type KafkaProducer struct {
	writer  *kafka.Writer
	brokers []string
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer(brokers []string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireOne,
		Async:                  true,                  // Enable async for better batching performance
		BatchSize:              100,                   // Number of messages per batch
		BatchBytes:             1048576,               // 1MB batch size
		BatchTimeout:           10 * time.Millisecond, // 10ms batch timeout
		WriteTimeout:           10 * time.Second,      // 10 second write timeout
		ReadTimeout:            10 * time.Second,      // 10 second read timeout
		AllowAutoTopicCreation: true,
	}

	return &KafkaProducer{
		writer:  writer,
		brokers: brokers,
	}
}

// Close closes the Kafka writer
func (kp *KafkaProducer) Close() error {
	return kp.writer.Close()
}

// SendEvent sends an event to Kafka
func (kp *KafkaProducer) SendEvent(event Event) error {
	// Generate topic name from domain, subdomain, and code
	topicName := fmt.Sprintf("%s_%s_%s", event.Domain, event.Subdomain, event.Code)

	// Convert event to JSON
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create a writer for this specific topic with batch and timeout settings
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(kp.brokers...),
		Topic:                  topicName,
		Balancer:               &kafka.LeastBytes{},
		BatchSize:              100,                   // Number of messages per batch
		BatchBytes:             1048576,               // 1MB batch size
		BatchTimeout:           10 * time.Millisecond, // 10ms batch timeout
		WriteTimeout:           10 * time.Second,      // 10 second write timeout
		ReadTimeout:            10 * time.Second,      // 10 second read timeout
		RequiredAcks:           kafka.RequireOne,
		Async:                  true, // Enable async for better batching
		AllowAutoTopicCreation: true,
	}
	defer writer.Close()

	// Create message (without Topic since writer already has it)
	message := kafka.Message{
		Key:   []byte(event.ID),
		Value: eventBytes,
		Time:  time.Now(),
	}

	// Create context with timeout for write operation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Send message with timeout context
	return writer.WriteMessages(ctx, message)
}

// SendEvents sends multiple events to Kafka in batches per topic
func (kp *KafkaProducer) SendEvents(events []Event) map[string]error {
	// Group events by topic
	eventsByTopic := make(map[string][]Event)
	errors := make(map[string]error)

	for _, event := range events {
		topicName := fmt.Sprintf("%s_%s_%s", event.Domain, event.Subdomain, event.Code)
		eventsByTopic[topicName] = append(eventsByTopic[topicName], event)
	}

	// Send events for each topic in batch
	for topicName, topicEvents := range eventsByTopic {
		// Create a writer for this specific topic
		writer := &kafka.Writer{
			Addr:                   kafka.TCP(kp.brokers...),
			Topic:                  topicName,
			Balancer:               &kafka.LeastBytes{},
			BatchSize:              100,                   // Number of messages per batch
			BatchBytes:             1048576,               // 1MB batch size
			BatchTimeout:           10 * time.Millisecond, // 10ms batch timeout
			WriteTimeout:           10 * time.Second,      // 10 second write timeout
			ReadTimeout:            10 * time.Second,      // 10 second read timeout
			RequiredAcks:           kafka.RequireOne,
			Async:                  true, // Enable async for better batching
			AllowAutoTopicCreation: true,
		}

		// Prepare messages for this topic
		messages := make([]kafka.Message, 0, len(topicEvents))
		for _, event := range topicEvents {
			eventBytes, err := json.Marshal(event)
			if err != nil {
				errors[event.ID] = fmt.Errorf("failed to marshal event: %w", err)
				continue
			}

			messages = append(messages, kafka.Message{
				Key:   []byte(event.ID),
				Value: eventBytes,
				Time:  time.Now(),
			})
		}

		// Create context with timeout for write operation
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// Send all messages for this topic in batch
		if err := writer.WriteMessages(ctx, messages...); err != nil {
			errors[topicName] = err
		}

		cancel()
		writer.Close()
	}

	return errors
}

// validateEvent validates the incoming event
func validateEvent(event Event) bool {
	if event.ID == "" || event.Domain == "" || event.Subdomain == "" || event.Code == "" {
		return false
	}
	return true
}

func main() {
	// Get port from environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default value
	}

	// Get Kafka brokers from environment variable
	brokersEnv := os.Getenv("KAFKA_BROKERS")
	if brokersEnv == "" {
		brokersEnv = "localhost:9092" // default value
	}
	brokers := strings.Split(brokersEnv, ",")

	// Create Kafka producer
	producer := NewKafkaProducer(brokers)
	defer producer.Close()

	// Create gin router
	r := gin.Default()

	// Health check endpoint
	r.GET("/protected/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// Events endpoint
	r.POST("/events", func(c *gin.Context) {
		var events []Event
		if err := c.ShouldBindJSON(&events); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid JSON format",
			})
			return
		}

		response := EventResponse{
			SuccessEventIds: []string{},
			InvalidEventIds: []string{},
			FailedEventIds:  []string{},
		}

		// Validate events first
		validEvents := []Event{}
		for _, event := range events {
			if !validateEvent(event) {
				response.InvalidEventIds = append(response.InvalidEventIds, event.ID)
				continue
			}
			validEvents = append(validEvents, event)
		}

		// Send valid events in batch
		if len(validEvents) > 0 {
			errors := producer.SendEvents(validEvents)

			// Process results
			for _, event := range validEvents {
				eventID := event.ID
				topicName := fmt.Sprintf("%s_%s_%s", event.Domain, event.Subdomain, event.Code)

				// Check for individual event errors (marshaling errors)
				if err, exists := errors[eventID]; exists {
					log.Printf("Error processing event with ID %s: %v", eventID, err)
					response.FailedEventIds = append(response.FailedEventIds, eventID)
				} else if err, exists := errors[topicName]; exists {
					// Check for topic-level errors (sending errors)
					log.Printf("Error sending event with ID %s to topic %s: %v", eventID, topicName, err)
					response.FailedEventIds = append(response.FailedEventIds, eventID)
				} else {
					response.SuccessEventIds = append(response.SuccessEventIds, eventID)
				}
			}
		}

		c.JSON(http.StatusOK, response)
	})

	// Convert port string to int for logging
	portInt, err := strconv.Atoi(port)
	if err != nil {
		log.Fatal("Invalid port number")
	}

	// Log startup information
	log.Printf("Starting server on port %d", portInt)
	log.Printf("Kafka brokers: %v", brokers)

	// Start server
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
