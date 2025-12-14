package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hospital-cqrs/internal/events"
	"hospital-cqrs/pkg/database"
	"hospital-cqrs/pkg/kafka"
)

func main() {
	log.Println("Iniciando Outbox Relay (Event Publisher - CQRS)...")

	// Conectar ao banco de dados
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	log.Println("Conectado ao PostgreSQL")

	// Criar producer Kafka
	producer, err := kafka.NewProducer()
	if err != nil {
		log.Fatalf("Erro ao criar producer Kafka: %v", err)
	}
	defer producer.Close()

	log.Println("Conectado ao Kafka")

	// Criar handler de eventos (para processar localmente)
	eventHandler := events.NewPrescricaoEventHandler(db)

	// Criar Outbox Relay
	relay := events.NewOutboxRelay(db, producer, eventHandler)

	log.Println("Outbox Relay configurado")
	log.Println("Verificando eventos pendentes a cada 2 segundos...")

	// Context para cancelamento
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Goroutine para executar o relay
	go func() {
		if err := relay.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("Erro no relay: %v", err)
		}
	}()

	// Goroutine para limpeza periódica (housekeeping)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Println("Executando limpeza de eventos processados...")
				if err := relay.LimparEventosProcessados(ctx, 7); err != nil {
					log.Printf("Erro na limpeza: %v", err)
				}
			}
		}
	}()

	// Goroutine para monitoramento
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count, err := relay.GetPendingCount(ctx)
				if err != nil {
					log.Printf("Erro ao obter contagem: %v", err)
					continue
				}
				if count > 0 {
					log.Printf("Eventos pendentes na Outbox: %d", count)
				}
			}
		}
	}()

	// Aguardar sinal de interrupção
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Encerrando Outbox Relay...")
	cancel()

	// Aguardar um pouco para processar eventos pendentes
	time.Sleep(2 * time.Second)
	log.Println("Outbox Relay encerrado")
}
