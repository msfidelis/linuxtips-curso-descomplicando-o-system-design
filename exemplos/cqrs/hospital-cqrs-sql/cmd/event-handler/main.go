package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"hospital-cqrs/internal/events"
	"hospital-cqrs/pkg/database"
	"hospital-cqrs/pkg/kafka"
)

func main() {
	log.Println("Iniciando Event Handler (Async Processor - CQRS)...")

	// Conectar ao banco de dados
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	// Criar handler de eventos
	eventHandler := events.NewPrescricaoEventHandler(db)

	// Criar consumidor Kafka
	consumer, err := kafka.NewConsumer("prescricoes", "")
	if err != nil {
		log.Fatalf("Erro ao criar consumidor Kafka: %v", err)
	}
	defer consumer.Close()

	log.Println("✓ Event Handler iniciado, aguardando eventos...")

	// Context para cancelamento
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Goroutine para consumir eventos
	go func() {
		if err := consumer.Consume(ctx, func(message []byte) error {
			return eventHandler.HandlePrescricaoCriada(ctx, message)
		}); err != nil && err != context.Canceled {
			log.Printf("Erro no consumidor: %v", err)
		}
	}()

	// Aguardar sinal de interrupção
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Encerrando Event Handler...")
	cancel()
}
