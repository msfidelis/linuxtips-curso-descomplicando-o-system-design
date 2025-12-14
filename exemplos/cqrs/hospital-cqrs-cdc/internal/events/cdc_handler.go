package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// DebeziumEvent representa um evento CDC do Debezium já "unwrapped" (ExtractNewRecordState)
// O payload vem diretamente com os dados, sem envelope before/after
type DebeziumEvent struct {
	// Campos do payload da tabela
	Data map[string]interface{}

	// Metadados do Debezium (injetados com prefixo __)
	Op       string `json:"__op"`      // c=create, u=update, d=delete, r=read
	Deleted  string `json:"__deleted"` // "true" ou "false"
	SourceMs int64  `json:"__source_ts_ms"`
}

// UnmarshalJSON customizado para capturar todos os campos
func (e *DebeziumEvent) UnmarshalJSON(data []byte) error {
	// Primeiro, deserializar para um map genérico
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extrair metadados do Debezium
	if op, ok := raw["__op"].(string); ok {
		e.Op = op
	}
	if deleted, ok := raw["__deleted"].(string); ok {
		e.Deleted = deleted
	}
	if sourceMs, ok := raw["__source_ts_ms"].(float64); ok {
		e.SourceMs = int64(sourceMs)
	}

	// Todos os campos (incluindo metadados) vão para Data
	e.Data = raw

	return nil
}

// CDCEventHandler processa eventos CDC do Debezium
type CDCEventHandler struct {
	db *sql.DB
}

// NewCDCEventHandler cria um novo handler de eventos CDC
func NewCDCEventHandler(db *sql.DB) *CDCEventHandler {
	return &CDCEventHandler{db: db}
}

// HandlePrescricaoCDC processa eventos CDC da tabela Prescricoes
func (h *CDCEventHandler) HandlePrescricaoCDC(ctx context.Context, eventData []byte) error {
	var event DebeziumEvent
	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("erro ao deserializar evento CDC: %w", err)
	}

	log.Printf("Evento CDC recebido: operação=%s", event.Op)

	// Processar apenas inserções (create) e reads (snapshot)
	if event.Op != "c" && event.Op != "r" {
		log.Printf("Ignorando operação %s - processamos apenas inserções", event.Op)
		return nil
	}

	// Verificar se foi deletado
	if event.Deleted == "true" {
		log.Printf("Ignorando registro deletado")
		return nil
	}

	// Extrair dados da prescrição do payload unwrapped
	idPrescricao := int(event.Data["id"].(float64))
	idMedico := int(event.Data["id_medico"].(float64))
	idPaciente := int(event.Data["id_paciente"].(float64))

	// Parse data_prescricao - pode vir como microsegundos (timestamp)
	var dataPrescricao time.Time
	switch v := event.Data["data_prescricao"].(type) {
	case float64:
		// Timestamp em microsegundos
		dataPrescricao = time.Unix(0, int64(v)*1000)
	case string:
		// String ISO 8601
		var err error
		dataPrescricao, err = time.Parse(time.RFC3339Nano, v)
		if err != nil {
			dataPrescricao, err = time.Parse("2006-01-02T15:04:05.999999Z07:00", v)
			if err != nil {
				return fmt.Errorf("erro ao parsear data_prescricao: %w", err)
			}
		}
	default:
		return fmt.Errorf("formato desconhecido para data_prescricao: %T", v)
	}

	log.Printf("Processando prescrição CDC: ID=%d Médico=%d Paciente=%d Data=%s",
		idPrescricao, idMedico, idPaciente, dataPrescricao.Format(time.RFC3339))

	// Buscar dados completos para popular as views
	medico, err := h.getMedico(ctx, idMedico)
	if err != nil {
		return fmt.Errorf("erro ao buscar médico: %w", err)
	}

	paciente, err := h.getPaciente(ctx, idPaciente)
	if err != nil {
		return fmt.Errorf("erro ao buscar paciente: %w", err)
	}

	// Buscar medicamentos da prescrição
	medicamentos, err := h.getMedicamentosPrescricao(ctx, idPrescricao)
	if err != nil {
		return fmt.Errorf("erro ao buscar medicamentos: %w", err)
	}

	// Atualizar views para cada medicamento
	for _, med := range medicamentos {
		// Atualizar View de Farmácia
		if err := h.atualizarViewFarmacia(ctx, idPrescricao, dataPrescricao, paciente, med); err != nil {
			log.Printf("Erro ao atualizar view farmácia: %v", err)
		}

		// Atualizar View de Prontuário
		if err := h.atualizarViewProntuario(ctx, idPrescricao, dataPrescricao, medico, paciente, med); err != nil {
			log.Printf("Erro ao atualizar view prontuário: %v", err)
		}
	}

	log.Printf("Evento CDC processado: Views atualizadas para prescrição %d", idPrescricao)
	return nil
}

// HandlePrescricaoMedicamentoCDC processa eventos CDC da tabela Prescricao_Medicamentos
func (h *CDCEventHandler) HandlePrescricaoMedicamentoCDC(ctx context.Context, eventData []byte) error {
	var event DebeziumEvent
	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("erro ao deserializar evento CDC: %w", err)
	}

	log.Printf("Evento CDC Medicamento recebido: operação=%s", event.Op)

	// Processar apenas inserções
	if event.Op != "c" && event.Op != "r" {
		log.Printf("Ignorando operação %s - processamos apenas inserções", event.Op)
		return nil
	}

	// Verificar se foi deletado
	if event.Deleted == "true" {
		log.Printf("Ignorando registro deletado")
		return nil
	}

	// Extrair dados do payload unwrapped
	idPrescricao := int(event.Data["id_prescricao"].(float64))
	idMedicamento := int(event.Data["id_medicamento"].(float64))
	horario := event.Data["horario"].(string)
	dosagem := event.Data["dosagem"].(string)

	log.Printf("Processando medicamento CDC: Prescrição=%d Medicamento=%d", idPrescricao, idMedicamento)

	// Buscar prescrição
	prescricao, err := h.getPrescricao(ctx, idPrescricao)
	if err != nil {
		return fmt.Errorf("erro ao buscar prescrição: %w", err)
	}

	// Buscar dados completos
	medico, err := h.getMedico(ctx, prescricao["id_medico"].(int))
	if err != nil {
		return fmt.Errorf("erro ao buscar médico: %w", err)
	}

	paciente, err := h.getPaciente(ctx, prescricao["id_paciente"].(int))
	if err != nil {
		return fmt.Errorf("erro ao buscar paciente: %w", err)
	}

	medicamento, err := h.getMedicamento(ctx, idMedicamento)
	if err != nil {
		return fmt.Errorf("erro ao buscar medicamento: %w", err)
	}

	// Adicionar horário e dosagem ao medicamento
	medicamento["horario"] = horario
	medicamento["dosagem"] = dosagem

	dataPrescricao := prescricao["data_prescricao"].(time.Time)

	// Atualizar views
	if err := h.atualizarViewFarmacia(ctx, idPrescricao, dataPrescricao, paciente, medicamento); err != nil {
		log.Printf("Erro ao atualizar view farmácia: %v", err)
	}

	if err := h.atualizarViewProntuario(ctx, idPrescricao, dataPrescricao, medico, paciente, medicamento); err != nil {
		log.Printf("Erro ao atualizar view prontuário: %v", err)
	}

	log.Printf("Medicamento CDC processado e views atualizadas")
	return nil
}

// getMedicamentosPrescricao busca todos os medicamentos de uma prescrição
func (h *CDCEventHandler) getMedicamentosPrescricao(ctx context.Context, idPrescricao int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			m.id, m.nome, m.descricao,
			pm.horario, pm.dosagem
		FROM Prescricao_Medicamentos pm
		JOIN Medicamentos m ON m.id = pm.id_medicamento
		WHERE pm.id_prescricao = $1
	`

	rows, err := h.db.QueryContext(ctx, query, idPrescricao)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var medicamentos []map[string]interface{}
	for rows.Next() {
		var id int
		var nome, descricao, horario, dosagem string

		if err := rows.Scan(&id, &nome, &descricao, &horario, &dosagem); err != nil {
			return nil, err
		}

		medicamentos = append(medicamentos, map[string]interface{}{
			"id":        id,
			"nome":      nome,
			"descricao": descricao,
			"horario":   horario,
			"dosagem":   dosagem,
		})
	}

	return medicamentos, nil
}

// getPrescricao busca uma prescrição por ID
func (h *CDCEventHandler) getPrescricao(ctx context.Context, id int) (map[string]interface{}, error) {
	var (
		prescricaoID   int
		idMedico       int
		idPaciente     int
		dataPrescricao time.Time
	)

	query := `SELECT id, id_medico, id_paciente, data_prescricao FROM Prescricoes WHERE id = $1`
	err := h.db.QueryRowContext(ctx, query, id).Scan(&prescricaoID, &idMedico, &idPaciente, &dataPrescricao)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":              prescricaoID,
		"id_medico":       idMedico,
		"id_paciente":     idPaciente,
		"data_prescricao": dataPrescricao,
	}, nil
}

// atualizarViewFarmacia atualiza o modelo de leitura da farmácia
func (h *CDCEventHandler) atualizarViewFarmacia(ctx context.Context, idPrescricao int, dataPrescricao time.Time, paciente, medicamento map[string]interface{}) error {
	query := `
		INSERT INTO View_Farmacia (
			id_prescricao, data_prescricao,
			paciente_id, paciente_nome, paciente_data_nascimento,
			medicamento_id, medicamento_nome, medicamento_descricao,
			horario, dosagem
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := h.db.ExecContext(ctx, query,
		idPrescricao, dataPrescricao,
		paciente["id"], paciente["nome"], paciente["data_nascimento"],
		medicamento["id"], medicamento["nome"], medicamento["descricao"],
		medicamento["horario"], medicamento["dosagem"],
	)

	if err != nil {
		return fmt.Errorf("erro ao inserir em View_Farmacia: %w", err)
	}

	log.Printf("View Farmácia atualizada para prescrição %d", idPrescricao)
	return nil
}

// atualizarViewProntuario atualiza o modelo de leitura do prontuário
func (h *CDCEventHandler) atualizarViewProntuario(ctx context.Context, idPrescricao int, dataPrescricao time.Time, medico, paciente, medicamento map[string]interface{}) error {
	query := `
		INSERT INTO View_Prontuario_Paciente (
			id_prescricao, data_prescricao,
			paciente_id, paciente_nome, paciente_data_nascimento, paciente_endereco,
			medico_id, medico_nome, medico_especialidade, medico_crm,
			medicamento_id, medicamento_nome, medicamento_descricao,
			horario, dosagem
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := h.db.ExecContext(ctx, query,
		idPrescricao, dataPrescricao,
		paciente["id"], paciente["nome"], paciente["data_nascimento"], paciente["endereco"],
		medico["id"], medico["nome"], medico["especialidade"], medico["crm"],
		medicamento["id"], medicamento["nome"], medicamento["descricao"],
		medicamento["horario"], medicamento["dosagem"],
	)

	if err != nil {
		return fmt.Errorf("erro ao inserir em View_Prontuario_Paciente: %w", err)
	}

	log.Printf("View Prontuário atualizada para prescrição %d", idPrescricao)
	return nil
}

// Funções auxiliares para buscar dados (reutilizadas do handler.go original)
func (h *CDCEventHandler) getMedico(ctx context.Context, id int) (map[string]interface{}, error) {
	var (
		medicoID      int
		nome          string
		especialidade string
		crm           string
	)

	query := `SELECT id, nome, especialidade, crm FROM Medicos WHERE id = $1`
	err := h.db.QueryRowContext(ctx, query, id).Scan(&medicoID, &nome, &especialidade, &crm)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":            medicoID,
		"nome":          nome,
		"especialidade": especialidade,
		"crm":           crm,
	}, nil
}

func (h *CDCEventHandler) getPaciente(ctx context.Context, id int) (map[string]interface{}, error) {
	var (
		pacienteID     int
		nome           string
		dataNascimento time.Time
		endereco       string
	)

	query := `SELECT id, nome, data_nascimento, endereco FROM Pacientes WHERE id = $1`
	err := h.db.QueryRowContext(ctx, query, id).Scan(&pacienteID, &nome, &dataNascimento, &endereco)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":              pacienteID,
		"nome":            nome,
		"data_nascimento": dataNascimento,
		"endereco":        endereco,
	}, nil
}

func (h *CDCEventHandler) getMedicamento(ctx context.Context, id int) (map[string]interface{}, error) {
	var (
		medicamentoID int
		nome          string
		descricao     string
	)

	query := `SELECT id, nome, descricao FROM Medicamentos WHERE id = $1`
	err := h.db.QueryRowContext(ctx, query, id).Scan(&medicamentoID, &nome, &descricao)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":        medicamentoID,
		"nome":      nome,
		"descricao": descricao,
	}, nil
}
