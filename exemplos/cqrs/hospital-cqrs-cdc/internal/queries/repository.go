package queries

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"hospital-cqrs/internal/domain"
)

// QueryRepository gerencia as consultas nos modelos de leitura (Read Side)
type QueryRepository struct {
	db *sql.DB
}

// NewQueryRepository cria um novo repositório de queries
func NewQueryRepository(db *sql.DB) *QueryRepository {
	return &QueryRepository{db: db}
}

// =========================================
// QUERIES - VIEW FARMÁCIA
// =========================================

// GetPrescricoesFarmacia retorna todas as prescrições para a farmácia
func (r *QueryRepository) GetPrescricoesFarmacia(ctx context.Context) ([]domain.PrescricaoFarmaciaDTO, error) {
	query := `
		SELECT 
			id_prescricao, data_prescricao,
			paciente_id, paciente_nome, paciente_data_nascimento,
			medicamento_id, medicamento_nome, medicamento_descricao,
			horario, dosagem
		FROM View_Farmacia
		ORDER BY data_prescricao DESC, id_prescricao, medicamento_nome
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar prescrições da farmácia: %w", err)
	}
	defer rows.Close()

	// Agrupar por prescrição
	prescricoesMap := make(map[int]*domain.PrescricaoFarmaciaDTO)

	for rows.Next() {
		var (
			idPrescricao           int
			dataPrescricao         time.Time
			pacienteID             int
			pacienteNome           string
			pacienteDataNascimento time.Time
			medicamentoID          int
			medicamentoNome        string
			medicamentoDescricao   string
			horario                string
			dosagem                string
		)

		if err := rows.Scan(&idPrescricao, &dataPrescricao, &pacienteID, &pacienteNome,
			&pacienteDataNascimento, &medicamentoID, &medicamentoNome,
			&medicamentoDescricao, &horario, &dosagem); err != nil {
			return nil, fmt.Errorf("erro ao scanear linha: %w", err)
		}

		// Se prescrição não existe no map, criar
		if _, exists := prescricoesMap[idPrescricao]; !exists {
			prescricoesMap[idPrescricao] = &domain.PrescricaoFarmaciaDTO{
				IDPrescricao:           idPrescricao,
				DataPrescricao:         dataPrescricao,
				PacienteID:             pacienteID,
				PacienteNome:           pacienteNome,
				PacienteDataNascimento: pacienteDataNascimento,
				Medicamentos:           []domain.MedicamentoFarmaciaDTO{},
			}
		}

		// Adicionar medicamento à prescrição
		medicamento := domain.MedicamentoFarmaciaDTO{
			MedicamentoID:        medicamentoID,
			MedicamentoNome:      medicamentoNome,
			MedicamentoDescricao: medicamentoDescricao,
			Horario:              horario,
			Dosagem:              dosagem,
		}
		prescricoesMap[idPrescricao].Medicamentos = append(prescricoesMap[idPrescricao].Medicamentos, medicamento)
	}

	// Converter map para slice
	prescricoes := make([]domain.PrescricaoFarmaciaDTO, 0, len(prescricoesMap))
	for _, p := range prescricoesMap {
		prescricoes = append(prescricoes, *p)
	}

	return prescricoes, nil
}

// GetPrescricaoFarmaciaByID retorna uma prescrição específica para a farmácia
func (r *QueryRepository) GetPrescricaoFarmaciaByID(ctx context.Context, idPrescricao int) (*domain.PrescricaoFarmaciaDTO, error) {
	query := `
		SELECT 
			id_prescricao, data_prescricao,
			paciente_id, paciente_nome, paciente_data_nascimento,
			medicamento_id, medicamento_nome, medicamento_descricao,
			horario, dosagem
		FROM View_Farmacia
		WHERE id_prescricao = $1
		ORDER BY medicamento_nome
	`

	rows, err := r.db.QueryContext(ctx, query, idPrescricao)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar prescrição: %w", err)
	}
	defer rows.Close()

	var prescricao *domain.PrescricaoFarmaciaDTO

	for rows.Next() {
		var (
			idPresc                int
			dataPresc              time.Time
			pacID                  int
			pacNome                string
			pacDataNasc            time.Time
			medID                  int
			medNome                string
			medDesc                string
			horario                string
			dosagem                string
		)

		if err := rows.Scan(&idPresc, &dataPresc, &pacID, &pacNome, &pacDataNasc,
			&medID, &medNome, &medDesc, &horario, &dosagem); err != nil {
			return nil, fmt.Errorf("erro ao scanear linha: %w", err)
		}

		if prescricao == nil {
			prescricao = &domain.PrescricaoFarmaciaDTO{
				IDPrescricao:           idPresc,
				DataPrescricao:         dataPresc,
				PacienteID:             pacID,
				PacienteNome:           pacNome,
				PacienteDataNascimento: pacDataNasc,
				Medicamentos:           []domain.MedicamentoFarmaciaDTO{},
			}
		}

		medicamento := domain.MedicamentoFarmaciaDTO{
			MedicamentoID:        medID,
			MedicamentoNome:      medNome,
			MedicamentoDescricao: medDesc,
			Horario:              horario,
			Dosagem:              dosagem,
		}
		prescricao.Medicamentos = append(prescricao.Medicamentos, medicamento)
	}

	if prescricao == nil {
		return nil, fmt.Errorf("prescrição não encontrada")
	}

	return prescricao, nil
}

// =========================================
// QUERIES - VIEW PRONTUÁRIO
// =========================================

// GetProntuarioPaciente retorna o prontuário completo de um paciente
func (r *QueryRepository) GetProntuarioPaciente(ctx context.Context, idPaciente int) (*domain.ProntuarioPacienteDTO, error) {
	query := `
		SELECT 
			id_prescricao, data_prescricao,
			paciente_id, paciente_nome, paciente_data_nascimento, paciente_endereco,
			medico_id, medico_nome, medico_especialidade, medico_crm,
			medicamento_id, medicamento_nome, medicamento_descricao,
			horario, dosagem
		FROM View_Prontuario_Paciente
		WHERE paciente_id = $1
		ORDER BY data_prescricao DESC, id_prescricao, medicamento_nome
	`

	rows, err := r.db.QueryContext(ctx, query, idPaciente)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar prontuário: %w", err)
	}
	defer rows.Close()

	var prontuario *domain.ProntuarioPacienteDTO
	prescricoesMap := make(map[int]*domain.PrescricaoProntuarioDTO)

	for rows.Next() {
		var (
			idPresc         int
			dataPresc       time.Time
			pacID           int
			pacNome         string
			pacDataNasc     time.Time
			pacEndereco     string
			medID           int
			medNome         string
			medEspec        string
			medCRM          string
			medicID         int
			medicNome       string
			medicDesc       string
			horario         string
			dosagem         string
		)

		if err := rows.Scan(&idPresc, &dataPresc, &pacID, &pacNome, &pacDataNasc, &pacEndereco,
			&medID, &medNome, &medEspec, &medCRM,
			&medicID, &medicNome, &medicDesc, &horario, &dosagem); err != nil {
			return nil, fmt.Errorf("erro ao scanear linha: %w", err)
		}

		// Criar prontuário se não existe
		if prontuario == nil {
			prontuario = &domain.ProntuarioPacienteDTO{
				PacienteID:             pacID,
				PacienteNome:           pacNome,
				PacienteDataNascimento: pacDataNasc,
				PacienteEndereco:       pacEndereco,
				Prescricoes:            []domain.PrescricaoProntuarioDTO{},
			}
		}

		// Se prescrição não existe no map, criar
		if _, exists := prescricoesMap[idPresc]; !exists {
			prescricoesMap[idPresc] = &domain.PrescricaoProntuarioDTO{
				IDPrescricao:        idPresc,
				DataPrescricao:      dataPresc,
				MedicoID:            medID,
				MedicoNome:          medNome,
				MedicoEspecialidade: medEspec,
				MedicoCRM:           medCRM,
				Medicamentos:        []domain.MedicamentoProntuarioDTO{},
			}
		}

		// Adicionar medicamento à prescrição
		medicamento := domain.MedicamentoProntuarioDTO{
			MedicamentoID:        medicID,
			MedicamentoNome:      medicNome,
			MedicamentoDescricao: medicDesc,
			Horario:              horario,
			Dosagem:              dosagem,
		}
		prescricoesMap[idPresc].Medicamentos = append(prescricoesMap[idPresc].Medicamentos, medicamento)
	}

	if prontuario == nil {
		return nil, fmt.Errorf("paciente não encontrado")
	}

	// Converter map para slice
	for _, p := range prescricoesMap {
		prontuario.Prescricoes = append(prontuario.Prescricoes, *p)
	}

	return prontuario, nil
}
