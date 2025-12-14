package commands

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"hospital-cqrs/internal/domain"
)

// PrescricaoRepository gerencia a persistência de prescrições (Write Side)
type PrescricaoRepository struct {
	db *sql.DB
}

// NewPrescricaoRepository cria um novo repositório de prescrições
func NewPrescricaoRepository(db *sql.DB) *PrescricaoRepository {
	return &PrescricaoRepository{db: db}
}

// CriarPrescricao cria uma nova prescrição no banco de dados
func (r *PrescricaoRepository) CriarPrescricao(ctx context.Context, dto domain.CriarPrescricaoDTO) (*domain.Prescricao, []domain.PrescricaoMedicamento, error) {
	// Iniciar transação
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer tx.Rollback()

	// Inserir prescrição
	var prescricao domain.Prescricao
	query := `
		INSERT INTO Prescricoes (id_medico, id_paciente, data_prescricao)
		VALUES ($1, $2, $3)
		RETURNING id, id_medico, id_paciente, data_prescricao, created_at
	`
	err = tx.QueryRowContext(ctx, query, dto.IDMedico, dto.IDPaciente, time.Now()).
		Scan(&prescricao.ID, &prescricao.IDMedico, &prescricao.IDPaciente, 
			&prescricao.DataPrescricao, &prescricao.CreatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao inserir prescrição: %w", err)
	}

	// Inserir medicamentos da prescrição
	var prescricaoMedicamentos []domain.PrescricaoMedicamento
	for _, med := range dto.Medicamentos {
		var pm domain.PrescricaoMedicamento
		queryMed := `
			INSERT INTO Prescricao_Medicamentos (id_prescricao, id_medicamento, horario, dosagem)
			VALUES ($1, $2, $3, $4)
			RETURNING id, id_prescricao, id_medicamento, horario, dosagem, created_at
		`
		err = tx.QueryRowContext(ctx, queryMed, prescricao.ID, med.IDMedicamento, med.Horario, med.Dosagem).
			Scan(&pm.ID, &pm.IDPrescricao, &pm.IDMedicamento, &pm.Horario, &pm.Dosagem, &pm.CreatedAt)
		if err != nil {
			return nil, nil, fmt.Errorf("erro ao inserir medicamento da prescrição: %w", err)
		}
		prescricaoMedicamentos = append(prescricaoMedicamentos, pm)
	}

	// Commit da transação
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("erro ao confirmar transação: %w", err)
	}

	return &prescricao, prescricaoMedicamentos, nil
}

// GetMedicoByID busca um médico por ID
func (r *PrescricaoRepository) GetMedicoByID(ctx context.Context, id int) (*domain.Medico, error) {
	var medico domain.Medico
	query := `SELECT id, nome, especialidade, crm FROM Medicos WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&medico.ID, &medico.Nome, &medico.Especialidade, &medico.CRM)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar médico: %w", err)
	}
	return &medico, nil
}

// GetPacienteByID busca um paciente por ID
func (r *PrescricaoRepository) GetPacienteByID(ctx context.Context, id int) (*domain.Paciente, error) {
	var paciente domain.Paciente
	query := `SELECT id, nome, data_nascimento, endereco FROM Pacientes WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&paciente.ID, &paciente.Nome, &paciente.DataNascimento, &paciente.Endereco)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar paciente: %w", err)
	}
	return &paciente, nil
}

// GetMedicamentoByID busca um medicamento por ID
func (r *PrescricaoRepository) GetMedicamentoByID(ctx context.Context, id int) (*domain.Medicamento, error) {
	var medicamento domain.Medicamento
	query := `SELECT id, nome, descricao FROM Medicamentos WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&medicamento.ID, &medicamento.Nome, &medicamento.Descricao)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar medicamento: %w", err)
	}
	return &medicamento, nil
}

// ListMedicos lista todos os médicos
func (r *PrescricaoRepository) ListMedicos(ctx context.Context) ([]domain.Medico, error) {
	query := `SELECT id, nome, especialidade, crm FROM Medicos ORDER BY nome`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar médicos: %w", err)
	}
	defer rows.Close()

	var medicos []domain.Medico
	for rows.Next() {
		var m domain.Medico
		if err := rows.Scan(&m.ID, &m.Nome, &m.Especialidade, &m.CRM); err != nil {
			return nil, fmt.Errorf("erro ao scanear médico: %w", err)
		}
		medicos = append(medicos, m)
	}
	return medicos, nil
}

// ListPacientes lista todos os pacientes
func (r *PrescricaoRepository) ListPacientes(ctx context.Context) ([]domain.Paciente, error) {
	query := `SELECT id, nome, data_nascimento, endereco FROM Pacientes ORDER BY nome`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar pacientes: %w", err)
	}
	defer rows.Close()

	var pacientes []domain.Paciente
	for rows.Next() {
		var p domain.Paciente
		if err := rows.Scan(&p.ID, &p.Nome, &p.DataNascimento, &p.Endereco); err != nil {
			return nil, fmt.Errorf("erro ao scanear paciente: %w", err)
		}
		pacientes = append(pacientes, p)
	}
	return pacientes, nil
}

// ListMedicamentos lista todos os medicamentos
func (r *PrescricaoRepository) ListMedicamentos(ctx context.Context) ([]domain.Medicamento, error) {
	query := `SELECT id, nome, descricao FROM Medicamentos ORDER BY nome`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar medicamentos: %w", err)
	}
	defer rows.Close()

	var medicamentos []domain.Medicamento
	for rows.Next() {
		var m domain.Medicamento
		if err := rows.Scan(&m.ID, &m.Nome, &m.Descricao); err != nil {
			return nil, fmt.Errorf("erro ao scanear medicamento: %w", err)
		}
		medicamentos = append(medicamentos, m)
	}
	return medicamentos, nil
}
