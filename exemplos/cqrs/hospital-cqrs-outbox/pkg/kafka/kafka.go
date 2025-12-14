package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/segmentio/kafka-go"
)

// Producer representa um produtor Kafka
type Producer struct {
	writer *kafka.Writer
}

// NewProducer cria um novo produtor Kafka
func NewProducer(topic ...string) (*Producer, error) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		return nil, fmt.Errorf("KAFKA_BROKERS não configurada")
	}

	// Topic padrão ou fornecido
	defaultTopic := "prescricoes"
	if len(topic) > 0 && topic[0] != "" {
		defaultTopic = topic[0]
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(strings.Split(brokers, ",")...),
		Topic:    defaultTopic,
		Balancer: &kafka.LeastBytes{},
	}

	log.Printf("Produtor Kafka criado para tópico: %s\n", defaultTopic)
	return &Producer{writer: writer}, nil
}

// Publish publica uma mensagem no Kafka
func (p *Producer) Publish(ctx context.Context, key string, value interface{}) error {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("erro ao serializar mensagem: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: valueBytes,
	})

	if err != nil {
		return fmt.Errorf("erro ao publicar mensagem: %w", err)
	}

	log.Printf("Mensagem publicada no tópico %s com key: %s\n", p.writer.Topic, key)
	return nil
}

// PublishRaw publica bytes brutos (já serializados) no Kafka
func (p *Producer) PublishRaw(ctx context.Context, key string, valueBytes []byte) error {
	err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: valueBytes,
	})

	if err != nil {
		return fmt.Errorf("erro ao publicar mensagem: %w", err)
	}

	log.Printf("Mensagem publicada no tópico %s com key: %s\n", p.writer.Topic, key)
	return nil
}

// GetTopic retorna o tópico configurado
func (p *Producer) GetTopic() string {
	return p.writer.Topic
}

// Close fecha o produtor
func (p *Producer) Close() error {
	return p.writer.Close()
}

// Consumer representa um consumidor Kafka
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer cria um novo consumidor Kafka
func NewConsumer(topic, groupID string) (*Consumer, error) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		return nil, fmt.Errorf("KAFKA_BROKERS não configurada")
	}

	if groupID == "" {
		groupID = os.Getenv("KAFKA_GROUP_ID")
		if groupID == "" {
			groupID = "default-group"
		}
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(brokers, ","),
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	log.Printf("Consumidor Kafka criado para tópico: %s (grupo: %s)\n", topic, groupID)
	return &Consumer{reader: reader}, nil
}

// Consume consome mensagens do Kafka
func (c *Consumer) Consume(ctx context.Context, handler func([]byte) error) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return fmt.Errorf("erro ao ler mensagem: %w", err)
		}

		log.Printf("Mensagem recebida do tópico %s: %s\n", c.reader.Config().Topic, string(msg.Key))

		if err := handler(msg.Value); err != nil {
			log.Printf("Erro ao processar mensagem: %v\n", err)
			// Continua processando outras mensagens
			continue
		}

		log.Println("Mensagem processada com sucesso")
	}
}

// Close fecha o consumidor
func (c *Consumer) Close() error {
	return c.reader.Close()
}
