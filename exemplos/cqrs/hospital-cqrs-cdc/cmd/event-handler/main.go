package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"hospital-cqrs/internal/events"

	"github.com/IBM/sarama"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("Iniciando Event Handler Service (CDC Mode)...")

	// Configuração do PostgreSQL
	dbHost := getEnv("DB_HOST", "postgres")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "hospital")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco: %v", err)
	}
	defer db.Close()

	// Testar conexão com retry
	for i := 0; i < 10; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		log.Printf("Aguardando banco de dados... tentativa %d/10", i+1)
		time.Sleep(3 * time.Second)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Não foi possível conectar ao banco: %v", err)
	}

	log.Println("Conectado ao PostgreSQL")

	// Configuração do Kafka
	kafkaBrokers := []string{getEnv("KAFKA_BROKERS", "localhost:9092")}
	groupID := getEnv("KAFKA_GROUP_ID", "event-handler-cdc-group")

	config := sarama.NewConfig()
	config.Version = sarama.V3_5_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true

	// Aguardar Kafka estar disponível
	for i := 0; i < 30; i++ {
		broker := sarama.NewBroker(kafkaBrokers[0])
		err := broker.Open(config)
		if err == nil {
			broker.Close()
			break
		}
		log.Printf("Aguardando Kafka... tentativa %d/30", i+1)
		time.Sleep(2 * time.Second)
	}

	consumerGroup, err := sarama.NewConsumerGroup(kafkaBrokers, groupID, config)
	if err != nil {
		log.Fatalf("Erro ao criar consumer group: %v", err)
	}
	defer consumerGroup.Close()

	log.Printf("Conectado ao Kafka: %v", kafkaBrokers)

	// Criar handler CDC
	cdcHandler := events.NewCDCEventHandler(db)
	consumer := &CDCConsumer{handler: cdcHandler}

	// Topics do Debezium (formato: server-name.schema.table)
	topics := []string{
		"hospital_db.public.prescricoes",
		"hospital_db.public.prescricao_medicamentos",
	}

	log.Printf("Consumindo tópicos CDC: %v", topics)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Consumer group em goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if err := consumerGroup.Consume(ctx, topics, consumer); err != nil {
				log.Printf("Erro no consumer group: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// Tratamento de erros do consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range consumerGroup.Errors() {
			log.Printf("❌ Erro no consumer: %v", err)
		}
	}()

	log.Println("Event Handler aguardando eventos CDC do Debezium...")

	// Graceful shutdown
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	<-sigterm

	log.Println("Encerrando Event Handler...")
	cancel()
	wg.Wait()
	log.Println("Event Handler encerrado")
}

// CDCConsumer implementa sarama.ConsumerGroupHandler para eventos CDC
type CDCConsumer struct {
	handler *events.CDCEventHandler
}

func (c *CDCConsumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *CDCConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *CDCConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		log.Printf("Evento CDC recebido do tópico %s (offset: %d)", message.Topic, message.Offset)

		var err error
		ctx := context.Background()

		// Roteamento baseado no tópico CDC
		switch message.Topic {
		case "hospital_db.public.prescricoes":
			err = c.handler.HandlePrescricaoCDC(ctx, message.Value)
		case "hospital_db.public.prescricao_medicamentos":
			err = c.handler.HandlePrescricaoMedicamentoCDC(ctx, message.Value)
		default:
			log.Printf("Tópico desconhecido: %s", message.Topic)
		}

		if err != nil {
			log.Printf("Erro ao processar evento CDC: %v", err)
			// Em produção, considere dead-letter queue aqui
		} else {
			session.MarkMessage(message, "")
		}
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
