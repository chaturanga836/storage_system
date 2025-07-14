package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"storage-engine/internal/config"
)

// Consumer handles consuming messages from various messaging backends
type Consumer interface {
	// Subscribe subscribes to one or more topics
	Subscribe(ctx context.Context, topics []string, handler MessageHandler) error

	// Consume starts consuming messages (blocking call)
	Consume(ctx context.Context) error

	// Close closes the consumer and cleans up resources
	Close() error
}

// MessageHandler handles consumed messages
type MessageHandler interface {
	Handle(ctx context.Context, message *Message) error
}

// MessageHandlerFunc adapts a function to implement MessageHandler
type MessageHandlerFunc func(ctx context.Context, message *Message) error

// Handle implements MessageHandler
func (f MessageHandlerFunc) Handle(ctx context.Context, message *Message) error {
	return f(ctx, message)
}

// ConsumerGroup represents a consumer group configuration
type ConsumerGroup struct {
	ID      string
	Topics  []string
	Handler MessageHandler
	Config  *ConsumerConfig
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	MaxRetries       int
	RetryBackoffMs   int
	SessionTimeoutMs int
	HeartbeatMs      int
	AutoCommit       bool
	AutoCommitMs     int
	BatchSize        int
	FetchTimeoutMs   int
}

// DefaultConsumerConfig returns default consumer configuration
func DefaultConsumerConfig() *ConsumerConfig {
	return &ConsumerConfig{
		MaxRetries:       3,
		RetryBackoffMs:   1000,
		SessionTimeoutMs: 30000,
		HeartbeatMs:      3000,
		AutoCommit:       true,
		AutoCommitMs:     5000,
		BatchSize:        100,
		FetchTimeoutMs:   1000,
	}
}

// KafkaConsumer implements Consumer for Apache Kafka
type KafkaConsumer struct {
	groupID      string
	topics       []string
	config       *config.KafkaConfig
	client       KafkaConsumerClient
	handler      MessageHandler
	deserializer MessageSerializer
	running      bool
	mu           sync.RWMutex
	stopChan     chan struct{}
	commitPolicy CommitPolicy
}

// KafkaConsumerClient interface for Kafka consumer operations
type KafkaConsumerClient interface {
	Subscribe(topics []string) error
	Poll(timeoutMs int) ([]ConsumedMessage, error)
	Commit(offsets map[string]int64) error
	Close() error
}

// ConsumedMessage represents a consumed message from Kafka
type ConsumedMessage struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   map[string][]byte
	Timestamp time.Time
}

// ToMessage converts ConsumedMessage to Message
func (cm *ConsumedMessage) ToMessage() (*Message, error) {
	headers := make(map[string]string)
	for k, v := range cm.Headers {
		headers[k] = string(v)
	}

	return &Message{
		ID:        fmt.Sprintf("%s-%d-%d", cm.Topic, cm.Partition, cm.Offset),
		Topic:     cm.Topic,
		Key:       string(cm.Key),
		Headers:   headers,
		Payload:   cm.Value,
		Timestamp: cm.Timestamp,
	}, nil
}

// NewKafkaConsumer creates a new Kafka consumer
func NewKafkaConsumer(groupID string, cfg *config.KafkaConfig) (*KafkaConsumer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("kafka config is required")
	}

	client, err := newKafkaConsumerClient(groupID, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer client: %w", err)
	}

	return &KafkaConsumer{
		groupID:      groupID,
		config:       cfg,
		client:       client,
		deserializer: &JSONSerializer{},
		stopChan:     make(chan struct{}),
		commitPolicy: &AutoCommitPolicy{
			IntervalMs: cfg.AutoCommitIntervalMs,
		},
	}, nil
}

// Subscribe subscribes to topics
func (kc *KafkaConsumer) Subscribe(ctx context.Context, topics []string, handler MessageHandler) error {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	if kc.running {
		return fmt.Errorf("consumer is already running")
	}

	kc.topics = topics
	kc.handler = handler

	return kc.client.Subscribe(topics)
}

// Consume starts consuming messages
func (kc *KafkaConsumer) Consume(ctx context.Context) error {
	kc.mu.Lock()
	if kc.running {
		kc.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	kc.running = true
	kc.mu.Unlock()

	defer func() {
		kc.mu.Lock()
		kc.running = false
		kc.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-kc.stopChan:
			return nil
		default:
			if err := kc.consumeBatch(ctx); err != nil {
				return fmt.Errorf("error consuming batch: %w", err)
			}
		}
	}
}

// consumeBatch consumes a batch of messages
func (kc *KafkaConsumer) consumeBatch(ctx context.Context) error {
	// Poll for messages
	consumedMessages, err := kc.client.Poll(kc.config.FetchTimeoutMs)
	if err != nil {
		return fmt.Errorf("failed to poll messages: %w", err)
	}

	if len(consumedMessages) == 0 {
		return nil // No messages available
	}

	// Process messages
	offsets := make(map[string]int64)
	for _, consumedMsg := range consumedMessages {
		message, err := consumedMsg.ToMessage()
		if err != nil {
			return fmt.Errorf("failed to convert consumed message: %w", err)
		}

		// Handle message with retry
		if err := kc.handleMessageWithRetry(ctx, message); err != nil {
			return fmt.Errorf("failed to handle message %s: %w", message.ID, err)
		}

		// Track offset for commit
		key := fmt.Sprintf("%s-%d", consumedMsg.Topic, consumedMsg.Partition)
		offsets[key] = consumedMsg.Offset + 1
	}

	// Commit offsets
	if err := kc.commitPolicy.ShouldCommit(); err == nil {
		if err := kc.client.Commit(offsets); err != nil {
			return fmt.Errorf("failed to commit offsets: %w", err)
		}
	}

	return nil
}

// handleMessageWithRetry handles a message with retry logic
func (kc *KafkaConsumer) handleMessageWithRetry(ctx context.Context, message *Message) error {
	maxRetries := kc.config.MaxRetries
	backoff := time.Duration(kc.config.RetryBackoffMs) * time.Millisecond

	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		if err := kc.handler.Handle(ctx, message); err != nil {
			lastErr = err

			if i < maxRetries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
					backoff *= 2 // Exponential backoff
					continue
				}
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// Close closes the Kafka consumer
func (kc *KafkaConsumer) Close() error {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	if kc.running {
		close(kc.stopChan)
	}

	return kc.client.Close()
}

// MemoryConsumer implements Consumer for in-memory testing
type MemoryConsumer struct {
	subscriptions map[string]MessageHandler
	messages      chan *Message
	running       bool
	mu            sync.RWMutex
	stopChan      chan struct{}
}

// NewMemoryConsumer creates a new in-memory consumer for testing
func NewMemoryConsumer() *MemoryConsumer {
	return &MemoryConsumer{
		subscriptions: make(map[string]MessageHandler),
		messages:      make(chan *Message, 1000),
		stopChan:      make(chan struct{}),
	}
}

// Subscribe subscribes to topics
func (mc *MemoryConsumer) Subscribe(ctx context.Context, topics []string, handler MessageHandler) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for _, topic := range topics {
		mc.subscriptions[topic] = handler
	}

	return nil
}

// Consume starts consuming messages
func (mc *MemoryConsumer) Consume(ctx context.Context) error {
	mc.mu.Lock()
	if mc.running {
		mc.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	mc.running = true
	mc.mu.Unlock()

	defer func() {
		mc.mu.Lock()
		mc.running = false
		mc.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-mc.stopChan:
			return nil
		case message := <-mc.messages:
			mc.mu.RLock()
			handler, exists := mc.subscriptions[message.Topic]
			mc.mu.RUnlock()

			if exists {
				if err := handler.Handle(ctx, message); err != nil {
					// In a real implementation, you might want to handle errors differently
					fmt.Printf("Error handling message %s: %v\n", message.ID, err)
				}
			}
		}
	}
}

// SendMessage sends a message to the consumer (for testing)
func (mc *MemoryConsumer) SendMessage(message *Message) {
	select {
	case mc.messages <- message:
	default:
		// Channel is full, drop message or handle differently
	}
}

// Close closes the memory consumer
func (mc *MemoryConsumer) Close() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.running {
		close(mc.stopChan)
	}

	return nil
}

// CommitPolicy defines when to commit offsets
type CommitPolicy interface {
	ShouldCommit() error
}

// AutoCommitPolicy commits offsets automatically at intervals
type AutoCommitPolicy struct {
	IntervalMs int
	lastCommit time.Time
	mu         sync.Mutex
}

// ShouldCommit returns nil if offsets should be committed
func (acp *AutoCommitPolicy) ShouldCommit() error {
	acp.mu.Lock()
	defer acp.mu.Unlock()

	now := time.Now()
	if now.Sub(acp.lastCommit) >= time.Duration(acp.IntervalMs)*time.Millisecond {
		acp.lastCommit = now
		return nil
	}

	return fmt.Errorf("not time to commit yet")
}

// EventConsumer provides high-level event consumption
type EventConsumer struct {
	consumer Consumer
	handlers map[EventType]EventHandler
	mu       sync.RWMutex
}

// EventHandler handles system events
type EventHandler interface {
	HandleEvent(ctx context.Context, event *Event) error
}

// EventHandlerFunc adapts a function to implement EventHandler
type EventHandlerFunc func(ctx context.Context, event *Event) error

// HandleEvent implements EventHandler
func (f EventHandlerFunc) HandleEvent(ctx context.Context, event *Event) error {
	return f(ctx, event)
}

// NewEventConsumer creates a new event consumer
func NewEventConsumer(consumer Consumer) *EventConsumer {
	return &EventConsumer{
		consumer: consumer,
		handlers: make(map[EventType]EventHandler),
	}
}

// RegisterHandler registers an event handler for a specific event type
func (ec *EventConsumer) RegisterHandler(eventType EventType, handler EventHandler) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.handlers[eventType] = handler
}

// Subscribe subscribes to events
func (ec *EventConsumer) Subscribe(ctx context.Context, eventTypes []EventType) error {
	topics := make([]string, len(eventTypes))
	for i, eventType := range eventTypes {
		topics[i] = string(eventType)
	}

	handler := MessageHandlerFunc(ec.handleMessage)
	return ec.consumer.Subscribe(ctx, topics, handler)
}

// handleMessage handles incoming messages and routes them to event handlers
func (ec *EventConsumer) handleMessage(ctx context.Context, message *Message) error {
	// Deserialize event from message
	var event Event
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		return fmt.Errorf("failed to deserialize event: %w", err)
	}

	// Find handler
	ec.mu.RLock()
	handler, exists := ec.handlers[event.Type]
	ec.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for event type: %s", event.Type)
	}

	// Handle event
	return handler.HandleEvent(ctx, &event)
}

// Consume starts consuming events
func (ec *EventConsumer) Consume(ctx context.Context) error {
	return ec.consumer.Consume(ctx)
}

// Close closes the event consumer
func (ec *EventConsumer) Close() error {
	return ec.consumer.Close()
}

// Mock Kafka consumer client implementation for compilation
func newKafkaConsumerClient(groupID string, cfg *config.KafkaConfig) (KafkaConsumerClient, error) {
	// In a real implementation, this would create an actual Kafka consumer client
	return &mockKafkaConsumerClient{}, nil
}

type mockKafkaConsumerClient struct{}

func (m *mockKafkaConsumerClient) Subscribe(topics []string) error {
	return nil
}

func (m *mockKafkaConsumerClient) Poll(timeoutMs int) ([]ConsumedMessage, error) {
	return []ConsumedMessage{}, nil
}

func (m *mockKafkaConsumerClient) Commit(offsets map[string]int64) error {
	return nil
}

func (m *mockKafkaConsumerClient) Close() error {
	return nil
}
