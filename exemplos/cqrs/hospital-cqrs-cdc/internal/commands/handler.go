package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"hospital-cqrs/internal/domain"
)

// PrescricaoHandler gerencia os comandos de prescrição
type PrescricaoHandler struct {
	repo *PrescricaoRepository
}

// NewPrescricaoHandler cria um novo handler de prescrições
// CDC: producer não é mais necessário - Debezium captura mudanças no banco
func NewPrescricaoHandler(db *sql.DB, _ interface{}) *PrescricaoHandler {
	return &PrescricaoHandler{
		repo: NewPrescricaoRepository(db),
	}
}

// CriarPrescricao processa o comando de criar prescrição
// CDC: Apenas persiste no banco - Debezium vai capturar a mudança automaticamente
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

	// Criar prescrição no banco de dados
	// Debezium vai capturar esta inserção e publicar no Kafka automaticamente
	prescricao, _, err := h.repo.CriarPrescricao(ctx, dto)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar prescrição: %w", err)
	}

	log.Printf("Prescrição criada com sucesso: ID %d (CDC vai capturar automaticamente)", prescricao.ID)
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
