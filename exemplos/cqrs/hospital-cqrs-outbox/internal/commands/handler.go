package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"hospital-cqrs/internal/domain"
	"hospital-cqrs/internal/events"
)

// PrescricaoHandler gerencia os comandos de prescrição (Outbox Pattern)
type PrescricaoHandler struct {
	repo *PrescricaoRepository
}

// NewPrescricaoHandler cria um novo handler de prescrições
func NewPrescricaoHandler(db *sql.DB) *PrescricaoHandler {
	return &PrescricaoHandler{
		repo: NewPrescricaoRepository(db),
	}
}

// CriarPrescricao processa o comando de criar prescrição usando Outbox Pattern
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

	// Criar prescrição e gravar evento na outbox (mesma transação)
	prescricao, _, err := h.repo.CriarPrescricaoComOutbox(ctx, dto)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar prescrição: %w", err)
	}

	log.Printf("Prescrição criada com sucesso (ID %d) e evento gravado na Outbox", prescricao.ID)
	log.Printf("Outbox Pattern: Evento será processado pelo relay assincronamente")

	return prescricao, nil
}

// criarEventoOutbox cria payload do evento para a outbox
func criarEventoOutbox(prescricao *domain.Prescricao, medicamentos []domain.PrescricaoMedicamento) ([]byte, error) {
	medicamentosEvent := make([]events.MedicamentoPrescritoEvent, len(medicamentos))
	for i, pm := range medicamentos {
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
	return json.Marshal(event)
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
