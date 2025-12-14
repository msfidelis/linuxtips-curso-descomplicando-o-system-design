package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"hospital-cqrs/internal/domain"
	"hospital-cqrs/internal/events"
	"hospital-cqrs/pkg/kafka"
)

// PrescricaoHandler gerencia os comandos de prescrição
type PrescricaoHandler struct {
	repo     *PrescricaoRepository
	producer *kafka.Producer
}

// NewPrescricaoHandler cria um novo handler de prescrições
func NewPrescricaoHandler(db *sql.DB, producer *kafka.Producer) *PrescricaoHandler {
	return &PrescricaoHandler{
		repo:     NewPrescricaoRepository(db),
		producer: producer,
	}
}

// CriarPrescricao processa o comando de criar prescrição
func (h *PrescricaoHandler) CriarPrescricao(ctx context.Context, dto domain.CriarPrescricaoDTO) (*domain.Prescricao, error) {
	// Validar se médico existe
	_, err := h.repo.GetMedicoByID(ctx, dto.IDMedico)
	if err != nil {
		return nil, fmt.Errorf("médico não encontrado: %w", err)
	}

	// Validar se paciente existe
	_, err = h.repo.GetPacienteByID(ctx, dto.IDPaciente)
	if err != nil {
		return nil, fmt.Errorf("paciente não encontrado: %w", err)
	}

	// Validar se medicamentos existem
	for _, med := range dto.Medicamentos {
		_, err := h.repo.GetMedicamentoByID(ctx, med.IDMedicamento)
		if err != nil {
			return nil, fmt.Errorf("medicamento %d não encontrado: %w", med.IDMedicamento, err)
		}
	}

	// Criar prescrição
	prescricao, prescricaoMedicamentos, err := h.repo.CriarPrescricao(ctx, dto)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar prescrição: %w", err)
	}

	// Criar evento de prescrição criada
	medicamentosEvent := make([]events.MedicamentoPrescritoEvent, len(prescricaoMedicamentos))
	for i, pm := range prescricaoMedicamentos {
		medicamentosEvent[i] = events.MedicamentoPrescritoEvent{
			IDMedicamento: pm.IDMedicamento,
			Horario:       pm.Horario,
			Dosagem:       pm.Dosagem,
		}
	}

	eventData := events.PrescricaoCriadaEventData{
		IDPrescricao:   prescricao.ID,
		IDMedico:       prescricao.IDMedico,
		IDPaciente:     prescricao.IDPaciente,
		DataPrescricao: prescricao.DataPrescricao,
		Medicamentos:   medicamentosEvent,
	}

	event := events.NewPrescricaoCriadaEvent(eventData)

	// Publicar evento no Kafka
	eventKey := fmt.Sprintf("prescricao-%d", prescricao.ID)
	if err := h.producer.Publish(ctx, eventKey, event); err != nil {
		log.Printf("AVISO: Erro ao publicar evento, mas prescrição foi criada: %v", err)
		// Não retorna erro pois a prescrição foi criada com sucesso
		// O evento poderá ser republicado posteriormente
	}

	log.Printf("✓ Prescrição criada com sucesso: ID %d", prescricao.ID)
	return prescricao, nil
}

// ListMedicos retorna a lista de médicos
func (h *PrescricaoHandler) ListMedicos(ctx context.Context) ([]domain.Medico, error) {
	return h.repo.ListMedicos(ctx)
}

// ListPacientes retorna a lista de pacientes
func (h *PrescricaoHandler) ListPacientes(ctx context.Context) ([]domain.Paciente, error) {
	return h.repo.ListPacientes(ctx)
}

// ListMedicamentos retorna a lista de medicamentos
func (h *PrescricaoHandler) ListMedicamentos(ctx context.Context) ([]domain.Medicamento, error) {
	return h.repo.ListMedicamentos(ctx)
}
