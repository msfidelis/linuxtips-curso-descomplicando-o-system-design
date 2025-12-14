package domain

import "time"

// =========================================
// COMMAND MODELS (Write Side)
// =========================================

// Medico representa um médico no sistema
type Medico struct {
	ID            int       `json:"id" db:"id"`
	Nome          string    `json:"nome" db:"nome"`
	Especialidade string    `json:"especialidade" db:"especialidade"`
	CRM           string    `json:"crm" db:"crm"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// Paciente representa um paciente no sistema
type Paciente struct {
	ID             int       `json:"id" db:"id"`
	Nome           string    `json:"nome" db:"nome"`
	DataNascimento time.Time `json:"data_nascimento" db:"data_nascimento"`
	Endereco       string    `json:"endereco" db:"endereco"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// Medicamento representa um medicamento disponível
type Medicamento struct {
	ID        int       `json:"id" db:"id"`
	Nome      string    `json:"nome" db:"nome"`
	Descricao string    `json:"descricao" db:"descricao"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Prescricao representa uma prescrição médica
type Prescricao struct {
	ID             int       `json:"id" db:"id"`
	IDMedico       int       `json:"id_medico" db:"id_medico"`
	IDPaciente     int       `json:"id_paciente" db:"id_paciente"`
	DataPrescricao time.Time `json:"data_prescricao" db:"data_prescricao"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// PrescricaoMedicamento representa a relação entre prescrição e medicamento
type PrescricaoMedicamento struct {
	ID            int       `json:"id" db:"id"`
	IDPrescricao  int       `json:"id_prescricao" db:"id_prescricao"`
	IDMedicamento int       `json:"id_medicamento" db:"id_medicamento"`
	Horario       string    `json:"horario" db:"horario"`
	Dosagem       string    `json:"dosagem" db:"dosagem"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// =========================================
// DTOs PARA COMANDOS
// =========================================

// MedicamentoPrescrito representa um medicamento em uma prescrição
type MedicamentoPrescrito struct {
	IDMedicamento int    `json:"id_medicamento" validate:"required"`
	Horario       string `json:"horario" validate:"required"`
	Dosagem       string `json:"dosagem" validate:"required"`
}

// CriarPrescricaoDTO é o DTO para criar uma nova prescrição
type CriarPrescricaoDTO struct {
	IDMedico     int                    `json:"id_medico" validate:"required"`
	IDPaciente   int                    `json:"id_paciente" validate:"required"`
	Medicamentos []MedicamentoPrescrito `json:"medicamentos" validate:"required,min=1"`
}

// =========================================
// QUERY MODELS (Read Side - Denormalized)
// =========================================

// ViewFarmacia representa o modelo de leitura otimizado para farmácia
type ViewFarmacia struct {
	ID                     int       `json:"id" db:"id"`
	IDPrescricao           int       `json:"id_prescricao" db:"id_prescricao"`
	DataPrescricao         time.Time `json:"data_prescricao" db:"data_prescricao"`
	PacienteID             int       `json:"paciente_id" db:"paciente_id"`
	PacienteNome           string    `json:"paciente_nome" db:"paciente_nome"`
	PacienteDataNascimento time.Time `json:"paciente_data_nascimento" db:"paciente_data_nascimento"`
	MedicamentoID          int       `json:"medicamento_id" db:"medicamento_id"`
	MedicamentoNome        string    `json:"medicamento_nome" db:"medicamento_nome"`
	MedicamentoDescricao   string    `json:"medicamento_descricao" db:"medicamento_descricao"`
	Horario                string    `json:"horario" db:"horario"`
	Dosagem                string    `json:"dosagem" db:"dosagem"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// ViewProntuarioPaciente representa o modelo de leitura otimizado para prontuário
type ViewProntuarioPaciente struct {
	ID                     int       `json:"id" db:"id"`
	IDPrescricao           int       `json:"id_prescricao" db:"id_prescricao"`
	DataPrescricao         time.Time `json:"data_prescricao" db:"data_prescricao"`
	PacienteID             int       `json:"paciente_id" db:"paciente_id"`
	PacienteNome           string    `json:"paciente_nome" db:"paciente_nome"`
	PacienteDataNascimento time.Time `json:"paciente_data_nascimento" db:"paciente_data_nascimento"`
	PacienteEndereco       string    `json:"paciente_endereco" db:"paciente_endereco"`
	MedicoID               int       `json:"medico_id" db:"medico_id"`
	MedicoNome             string    `json:"medico_nome" db:"medico_nome"`
	MedicoEspecialidade    string    `json:"medico_especialidade" db:"medico_especialidade"`
	MedicoCRM              string    `json:"medico_crm" db:"medico_crm"`
	MedicamentoID          int       `json:"medicamento_id" db:"medicamento_id"`
	MedicamentoNome        string    `json:"medicamento_nome" db:"medicamento_nome"`
	MedicamentoDescricao   string    `json:"medicamento_descricao" db:"medicamento_descricao"`
	Horario                string    `json:"horario" db:"horario"`
	Dosagem                string    `json:"dosagem" db:"dosagem"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// =========================================
// DTOs PARA QUERIES
// =========================================

// PrescricaoFarmaciaDTO agrupa prescrições para a farmácia
type PrescricaoFarmaciaDTO struct {
	IDPrescricao           int                      `json:"id_prescricao"`
	DataPrescricao         time.Time                `json:"data_prescricao"`
	PacienteID             int                      `json:"paciente_id"`
	PacienteNome           string                   `json:"paciente_nome"`
	PacienteDataNascimento time.Time                `json:"paciente_data_nascimento"`
	Medicamentos           []MedicamentoFarmaciaDTO `json:"medicamentos"`
}

// MedicamentoFarmaciaDTO representa medicamento para farmácia
type MedicamentoFarmaciaDTO struct {
	MedicamentoID        int    `json:"medicamento_id"`
	MedicamentoNome      string `json:"medicamento_nome"`
	MedicamentoDescricao string `json:"medicamento_descricao"`
	Horario              string `json:"horario"`
	Dosagem              string `json:"dosagem"`
}

// ProntuarioPacienteDTO agrupa prescrições do prontuário
type ProntuarioPacienteDTO struct {
	PacienteID             int                       `json:"paciente_id"`
	PacienteNome           string                    `json:"paciente_nome"`
	PacienteDataNascimento time.Time                 `json:"paciente_data_nascimento"`
	PacienteEndereco       string                    `json:"paciente_endereco"`
	Prescricoes            []PrescricaoProntuarioDTO `json:"prescricoes"`
}

// PrescricaoProntuarioDTO representa uma prescrição no prontuário
type PrescricaoProntuarioDTO struct {
	IDPrescricao        int                        `json:"id_prescricao"`
	DataPrescricao      time.Time                  `json:"data_prescricao"`
	MedicoID            int                        `json:"medico_id"`
	MedicoNome          string                     `json:"medico_nome"`
	MedicoEspecialidade string                     `json:"medico_especialidade"`
	MedicoCRM           string                     `json:"medico_crm"`
	Medicamentos        []MedicamentoProntuarioDTO `json:"medicamentos"`
}

// MedicamentoProntuarioDTO representa medicamento no prontuário
type MedicamentoProntuarioDTO struct {
	MedicamentoID        int    `json:"medicamento_id"`
	MedicamentoNome      string `json:"medicamento_nome"`
	MedicamentoDescricao string `json:"medicamento_descricao"`
	Horario              string `json:"horario"`
	Dosagem              string `json:"dosagem"`
}
