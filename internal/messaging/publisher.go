package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/storage-system/internal/common"
	"github.com/storage-system/internal/config"
)

// Publisher handles publishing events to various messaging backends
type Publisher interface {
	// Publish sends a message to the specified topic
	Publish(ctx context.Context, topic string, message *Message) error
	
	// PublishBatch sends multiple messages to the specified topic
	PublishBatch(ctx context.Context, topic string, messages []*Message) error
	
	// Close closes the publisher and cleans up resources
	Close() error
}

// Message represents a message to be published
type Message struct {
	ID        string                 `json:"id"`
	Topic     string                 `json:"topic"`
	Key       string                 `json:"key,omitempty"`
	Headers   map[string]string      `json:"headers,omitempty"`
	Payload   []byte                 `json:"payload"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Retry     int                    `json:"retry"`
}

// EventType represents different types of events in the system
type EventType string

const (
	EventDataIngested     EventType = "data.ingested"
	EventDataProcessed    EventType = "data.processed"
	EventCompactionStart  EventType = "compaction.start"
	EventCompactionEnd    EventType = "compaction.end"
	EventIndexUpdated     EventType = "index.updated"
	EventSchemaEvolved    EventType = "schema.evolved"
	EventTableCreated     EventType = "table.created"
	EventTableDropped     EventType = "table.dropped"
	EventQueryExecuted    EventType = "query.executed"
	EventErrorOccurred    EventType = "error.occurred"
	EventMetricsUpdated   EventType = "metrics.updated"
)

// Event represents a system event
type Event struct {
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	TraceID   string                 `json:"trace_id,omitempty"`
}

// ToMessage converts an Event to a Message
func (e *Event) ToMessage() (*Message, error) {
	payload, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	return &Message{
		ID:        common.GenerateID(),
		Topic:     string(e.Type),
		Key:       e.Source,
		Payload:   payload,
		Timestamp: e.Timestamp,
		Headers: map[string]string{
			"event-type": string(e.Type),
			"source":     e.Source,
			"trace-id":   e.TraceID,
		},
	}, nil
}

// KafkaPublisher implements Publisher for Apache Kafka
type KafkaPublisher struct {
	brokers       []string
	config        *config.KafkaConfig
	client        KafkaClient // Interface for actual Kafka client
	retryPolicy   *RetryPolicy
	serializer    MessageSerializer
	mu            sync.RWMutex
	closed        bool
}

// KafkaClient interface for Kafka operations (abstraction for testing)
type KafkaClient interface {
	Produce(topic string, key []byte, value []byte, headers map[string][]byte) error
	ProduceBatch(topic string, messages []KafkaMessage) error
	Close() error
}

// KafkaMessage represents a Kafka message
type KafkaMessage struct {
	Key     []byte
	Value   []byte
	Headers map[string][]byte
}

// NewKafkaPublisher creates a new Kafka publisher
func NewKafkaPublisher(cfg *config.KafkaConfig) (*KafkaPublisher, error) {
	if cfg == nil {
		return nil, fmt.Errorf("kafka config is required")
	}

	// Initialize Kafka client based on config
	client, err := newKafkaClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	return &KafkaPublisher{
		brokers: cfg.Brokers,
		config:  cfg,
		client:  client,
		retryPolicy: &RetryPolicy{
			MaxRetries:  cfg.MaxRetries,
			BackoffBase: time.Duration(cfg.RetryBackoffMs) * time.Millisecond,
			MaxBackoff:  time.Duration(cfg.MaxRetryBackoffMs) * time.Millisecond,
		},
		serializer: &JSONSerializer{},
	}, nil
}

// Publish publishes a single message to Kafka
func (kp *KafkaPublisher) Publish(ctx context.Context, topic string, message *Message) error {
	kp.mu.RLock()
	if kp.closed {
		kp.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	kp.mu.RUnlock()

	// Serialize message
	value, err := kp.serializer.Serialize(message)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Convert headers
	headers := make(map[string][]byte)
	for k, v := range message.Headers {
		headers[k] = []byte(v)
	}

	// Publish with retry
	return kp.retryPolicy.Execute(ctx, func() error {
		return kp.client.Produce(topic, []byte(message.Key), value, headers)
	})
}

// PublishBatch publishes multiple messages to Kafka
func (kp *KafkaPublisher) PublishBatch(ctx context.Context, topic string, messages []*Message) error {
	kp.mu.RLock()
	if kp.closed {
		kp.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	kp.mu.RUnlock()

	kafkaMessages := make([]KafkaMessage, 0, len(messages))
	
	for _, msg := range messages {
		value, err := kp.serializer.Serialize(msg)
		if err != nil {
			return fmt.Errorf("failed to serialize message %s: %w", msg.ID, err)
		}

		headers := make(map[string][]byte)
		for k, v := range msg.Headers {
			headers[k] = []byte(v)
		}

		kafkaMessages = append(kafkaMessages, KafkaMessage{
			Key:     []byte(msg.Key),
			Value:   value,
			Headers: headers,
		})
	}

	return kp.retryPolicy.Execute(ctx, func() error {
		return kp.client.ProduceBatch(topic, kafkaMessages)
	})
}

// Close closes the Kafka publisher
func (kp *KafkaPublisher) Close() error {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	if kp.closed {
		return nil
	}

	kp.closed = true
	return kp.client.Close()
}

// MemoryPublisher implements Publisher for in-memory testing
type MemoryPublisher struct {
	messages map[string][]*Message
	mu       sync.RWMutex
}

// NewMemoryPublisher creates a new in-memory publisher for testing
func NewMemoryPublisher() *MemoryPublisher {
	return &MemoryPublisher{
		messages: make(map[string][]*Message),
	}
}

// Publish publishes a message to memory
func (mp *MemoryPublisher) Publish(ctx context.Context, topic string, message *Message) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.messages[topic] == nil {
		mp.messages[topic] = make([]*Message, 0)
	}
	
	mp.messages[topic] = append(mp.messages[topic], message)
	return nil
}

// PublishBatch publishes multiple messages to memory
func (mp *MemoryPublisher) PublishBatch(ctx context.Context, topic string, messages []*Message) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.messages[topic] == nil {
		mp.messages[topic] = make([]*Message, 0)
	}
	
	mp.messages[topic] = append(mp.messages[topic], messages...)
	return nil
}

// GetMessages returns all messages for a topic (for testing)
func (mp *MemoryPublisher) GetMessages(topic string) []*Message {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	messages := mp.messages[topic]
	if messages == nil {
		return nil
	}

	// Return a copy
	result := make([]*Message, len(messages))
	copy(result, messages)
	return result
}

// Clear clears all messages (for testing)
func (mp *MemoryPublisher) Clear() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.messages = make(map[string][]*Message)
}

// Close closes the memory publisher
func (mp *MemoryPublisher) Close() error {
	mp.Clear()
	return nil
}

// RetryPolicy defines retry behavior for failed publishes
type RetryPolicy struct {
	MaxRetries  int
	BackoffBase time.Duration
	MaxBackoff  time.Duration
}

// Execute executes a function with retry logic
func (rp *RetryPolicy) Execute(ctx context.Context, fn func() error) error {
	var lastErr error
	
	for i := 0; i <= rp.MaxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err
			
			if i < rp.MaxRetries {
				backoff := rp.calculateBackoff(i)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
					continue
				}
			}
		} else {
			return nil
		}
	}
	
	return fmt.Errorf("failed after %d retries: %w", rp.MaxRetries, lastErr)
}

// calculateBackoff calculates exponential backoff with jitter
func (rp *RetryPolicy) calculateBackoff(attempt int) time.Duration {
	backoff := rp.BackoffBase * time.Duration(1<<uint(attempt))
	if backoff > rp.MaxBackoff {
		backoff = rp.MaxBackoff
	}
	return backoff
}

// MessageSerializer interface for message serialization
type MessageSerializer interface {
	Serialize(message *Message) ([]byte, error)
	Deserialize(data []byte) (*Message, error)
}

// JSONSerializer implements MessageSerializer using JSON
type JSONSerializer struct{}

// Serialize serializes a message to JSON
func (js *JSONSerializer) Serialize(message *Message) ([]byte, error) {
	return json.Marshal(message)
}

// Deserialize deserializes JSON data to a message
func (js *JSONSerializer) Deserialize(data []byte) (*Message, error) {
	var message Message
	err := json.Unmarshal(data, &message)
	return &message, err
}

// EventPublisher provides high-level event publishing
type EventPublisher struct {
	publisher Publisher
	source    string
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(publisher Publisher, source string) *EventPublisher {
	return &EventPublisher{
		publisher: publisher,
		source:    source,
	}
}

// PublishEvent publishes a system event
func (ep *EventPublisher) PublishEvent(ctx context.Context, eventType EventType, data map[string]interface{}) error {
	event := &Event{
		Type:      eventType,
		Source:    ep.source,
		Data:      data,
		Timestamp: time.Now(),
		TraceID:   common.GetTraceID(ctx),
	}

	message, err := event.ToMessage()
	if err != nil {
		return fmt.Errorf("failed to convert event to message: %w", err)
	}

	return ep.publisher.Publish(ctx, string(eventType), message)
}

// Mock Kafka client implementation for compilation
func newKafkaClient(cfg *config.KafkaConfig) (KafkaClient, error) {
	// In a real implementation, this would create an actual Kafka client
	// For now, return a mock implementation
	return &mockKafkaClient{}, nil
}

type mockKafkaClient struct{}

func (m *mockKafkaClient) Produce(topic string, key []byte, value []byte, headers map[string][]byte) error {
	return nil
}

func (m *mockKafkaClient) ProduceBatch(topic string, messages []KafkaMessage) error {
	return nil
}

func (m *mockKafkaClient) Close() error {
	return nil
}
