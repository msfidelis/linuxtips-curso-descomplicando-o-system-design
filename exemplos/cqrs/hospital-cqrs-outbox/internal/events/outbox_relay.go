package events

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"hospital-cqrs/pkg/kafka"
)

// OutboxEvent representa um evento armazenado na tabela Outbox
type OutboxEvent struct {
	ID            int64
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte
	CreatedAt     time.Time
	ProcessedAt   *time.Time
	PublishedAt   *time.Time
	ErrorMessage  *string
	RetryCount    int
}

// OutboxRelay é responsável por ler eventos da outbox e publicá-los no Kafka
type OutboxRelay struct {
	db             *sql.DB
	producer       *kafka.Producer
	pollInterval   time.Duration
	batchSize      int
	maxRetries     int
	eventProcessor *PrescricaoEventHandler
}

// NewOutboxRelay cria um novo relay de outbox
func NewOutboxRelay(db *sql.DB, producer *kafka.Producer, eventProcessor *PrescricaoEventHandler) *OutboxRelay {
	return &OutboxRelay{
		db:             db,
		producer:       producer,
		pollInterval:   2 * time.Second, // Verifica outbox a cada 2 segundos
		batchSize:      100,             // Processa até 100 eventos por vez
		maxRetries:     5,               // Máximo 5 tentativas por evento
		eventProcessor: eventProcessor,
	}
}

// Start inicia o relay em loop contínuo
func (r *OutboxRelay) Start(ctx context.Context) error {
	log.Println("Outbox Relay iniciado - monitorando eventos pendentes...")

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Outbox Relay encerrando...")
			return ctx.Err()
		case <-ticker.C:
			if err := r.processarEventosPendentes(ctx); err != nil {
				log.Printf("Erro ao processar eventos pendentes: %v", err)
			}
		}
	}
}

// processarEventosPendentes busca e processa eventos não processados
func (r *OutboxRelay) processarEventosPendentes(ctx context.Context) error {
	// Buscar eventos pendentes (processed_at IS NULL)
	eventos, err := r.buscarEventosPendentes(ctx)
	if err != nil {
		return fmt.Errorf("erro ao buscar eventos pendentes: %w", err)
	}

	if len(eventos) == 0 {
		return nil // Nenhum evento pendente
	}

	log.Printf("Processando %d evento(s) da Outbox...", len(eventos))

	for _, evento := range eventos {
		if err := r.processarEvento(ctx, evento); err != nil {
			log.Printf("Erro ao processar evento %d: %v", evento.ID, err)

			// Marcar erro na outbox
			if err := r.marcarErro(ctx, evento.ID, err.Error()); err != nil {
				log.Printf("Erro ao marcar falha do evento %d: %v", evento.ID, err)
			}
		}
	}

	return nil
}

// buscarEventosPendentes retorna eventos não processados
func (r *OutboxRelay) buscarEventosPendentes(ctx context.Context) ([]OutboxEvent, error) {
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, 
		       created_at, processed_at, published_at, error_message, retry_count
		FROM Outbox_Events
		WHERE processed_at IS NULL 
		  AND retry_count < $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, r.maxRetries, r.batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eventos []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		err := rows.Scan(
			&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload,
			&e.CreatedAt, &e.ProcessedAt, &e.PublishedAt, &e.ErrorMessage, &e.RetryCount,
		)
		if err != nil {
			return nil, err
		}
		eventos = append(eventos, e)
	}

	return eventos, nil
}

// processarEvento processa um único evento da outbox
func (r *OutboxRelay) processarEvento(ctx context.Context, evento OutboxEvent) error {
	log.Printf("Processando evento Outbox: ID=%d Tipo=%s Agregado=%s/%s",
		evento.ID, evento.EventType, evento.AggregateType, evento.AggregateID)

	// 1. Publicar evento no Kafka
	eventKey := fmt.Sprintf("%s-%s", evento.AggregateType, evento.AggregateID)
	if err := r.producer.PublishRaw(ctx, eventKey, evento.Payload); err != nil {
		return fmt.Errorf("erro ao publicar no Kafka: %w", err)
	}

	publishedAt := time.Now()
	log.Printf("Evento %d publicado no Kafka (topic: %s, key: %s)",
		evento.ID, r.producer.GetTopic(), eventKey)

	// 2. Processar evento localmente para atualizar views
	// (Isso simula o consumidor que normalmente rodaria em outro serviço)
	if err := r.processarEventoLocal(ctx, evento); err != nil {
		log.Printf("Erro ao processar evento localmente (views): %v", err)
		// Não falha a publicação se view falhar - apenas loga
	}

	// 3. Marcar como processado na outbox
	if err := r.marcarProcessado(ctx, evento.ID, publishedAt); err != nil {
		return fmt.Errorf("erro ao marcar evento como processado: %w", err)
	}

	return nil
}

// processarEventoLocal processa o evento para atualizar as views
func (r *OutboxRelay) processarEventoLocal(ctx context.Context, evento OutboxEvent) error {
	switch evento.EventType {
	case "prescricao.criada":
		return r.eventProcessor.HandlePrescricaoCriada(ctx, evento.Payload)
	default:
		log.Printf("Tipo de evento desconhecido: %s", evento.EventType)
		return nil
	}
}

// marcarProcessado marca evento como processado
func (r *OutboxRelay) marcarProcessado(ctx context.Context, eventoID int64, publishedAt time.Time) error {
	query := `
		UPDATE Outbox_Events
		SET processed_at = $1, published_at = $2, error_message = NULL
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, time.Now(), publishedAt, eventoID)
	if err != nil {
		return err
	}

	log.Printf("Evento %d marcado como processado", eventoID)
	return nil
}

// marcarErro marca evento com erro e incrementa retry_count
func (r *OutboxRelay) marcarErro(ctx context.Context, eventoID int64, errorMsg string) error {
	query := `
		UPDATE Outbox_Events
		SET retry_count = retry_count + 1, error_message = $1
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, errorMsg, eventoID)
	return err
}

// LimparEventosProcessados remove eventos antigos já processados (housekeeping)
func (r *OutboxRelay) LimparEventosProcessados(ctx context.Context, retentionDays int) error {
	query := `
		DELETE FROM Outbox_Events
		WHERE processed_at IS NOT NULL
		  AND processed_at < NOW() - INTERVAL '1 day' * $1
	`
	result, err := r.db.ExecContext(ctx, query, retentionDays)
	if err != nil {
		return err
	}

	rowsDeleted, _ := result.RowsAffected()
	if rowsDeleted > 0 {
		log.Printf("🧹 Limpeza: %d evento(s) antigo(s) removido(s) da Outbox", rowsDeleted)
	}

	return nil
}

// GetPendingCount retorna quantidade de eventos pendentes
func (r *OutboxRelay) GetPendingCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM Outbox_Events WHERE processed_at IS NULL`
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}
