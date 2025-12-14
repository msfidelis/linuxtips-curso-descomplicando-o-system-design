package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// PrescricaoEventHandler processa eventos de prescrição
type PrescricaoEventHandler struct {
	db *sql.DB
}

// NewPrescricaoEventHandler cria um novo handler de eventos
func NewPrescricaoEventHandler(db *sql.DB) *PrescricaoEventHandler {
	return &PrescricaoEventHandler{db: db}
}

// HandlePrescricaoCriada processa o evento de prescrição criada
func (h *PrescricaoEventHandler) HandlePrescricaoCriada(ctx context.Context, eventData []byte) error {
	var event Event
	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("erro ao deserializar evento: %w", err)
	}

	if event.Type != PrescricaoCriadaEvent {
		return fmt.Errorf("tipo de evento inválido: %s", event.Type)
	}

	// Extrair dados do evento
	idPrescricao := int(event.Data["id_prescricao"].(float64))
	idMedico := int(event.Data["id_medico"].(float64))
	idPaciente := int(event.Data["id_paciente"].(float64))
	dataPrescricaoStr := event.Data["data_prescricao"].(string)
	dataPrescricao, _ := time.Parse(time.RFC3339, dataPrescricaoStr)

	medicamentosData := event.Data["medicamentos"].([]interface{})

	log.Printf("📨 Processando evento: Prescrição %d criada", idPrescricao)

	// Buscar dados completos para popular as views
	medico, err := h.getMedico(ctx, idMedico)
	if err != nil {
		return fmt.Errorf("erro ao buscar médico: %w", err)
	}

	paciente, err := h.getPaciente(ctx, idPaciente)
	if err != nil {
		return fmt.Errorf("erro ao buscar paciente: %w", err)
	}

	// Processar cada medicamento e atualizar as views
	for _, medData := range medicamentosData {
		medMap := medData.(map[string]interface{})
		idMedicamento := int(medMap["id_medicamento"].(float64))
		horario := medMap["horario"].(string)
		dosagem := medMap["dosagem"].(string)

		medicamento, err := h.getMedicamento(ctx, idMedicamento)
		if err != nil {
			return fmt.Errorf("erro ao buscar medicamento: %w", err)
		}

		// Atualizar View de Farmácia
		if err := h.atualizarViewFarmacia(ctx, idPrescricao, dataPrescricao, paciente, medicamento, horario, dosagem); err != nil {
			log.Printf("Erro ao atualizar view farmácia: %v", err)
			// Continua para tentar atualizar próxima view
		}

		// Atualizar View de Prontuário
		if err := h.atualizarViewProntuario(ctx, idPrescricao, dataPrescricao, medico, paciente, medicamento, horario, dosagem); err != nil {
			log.Printf("Erro ao atualizar view prontuário: %v", err)
		}
	}

	log.Printf("✓ Evento processado: Views atualizadas para prescrição %d", idPrescricao)
	return nil
}

// atualizarViewFarmacia atualiza o modelo de leitura da farmácia
func (h *PrescricaoEventHandler) atualizarViewFarmacia(ctx context.Context, idPrescricao int, dataPrescricao time.Time, paciente, medicamento map[string]interface{}, horario, dosagem string) error {
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
		horario, dosagem,
	)

	if err != nil {
		return fmt.Errorf("erro ao inserir em View_Farmacia: %w", err)
	}

	log.Printf("✓ View Farmácia atualizada para prescrição %d", idPrescricao)
	return nil
}

// atualizarViewProntuario atualiza o modelo de leitura do prontuário
func (h *PrescricaoEventHandler) atualizarViewProntuario(ctx context.Context, idPrescricao int, dataPrescricao time.Time, medico, paciente, medicamento map[string]interface{}, horario, dosagem string) error {
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
		horario, dosagem,
	)

	if err != nil {
		return fmt.Errorf("erro ao inserir em View_Prontuario_Paciente: %w", err)
	}

	log.Printf("✓ View Prontuário atualizada para prescrição %d", idPrescricao)
	return nil
}

// Funções auxiliares para buscar dados
func (h *PrescricaoEventHandler) getMedico(ctx context.Context, id int) (map[string]interface{}, error) {
	var (
		medicoID          int
		nome              string
		especialidade     string
		crm               string
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

func (h *PrescricaoEventHandler) getPaciente(ctx context.Context, id int) (map[string]interface{}, error) {
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

func (h *PrescricaoEventHandler) getMedicamento(ctx context.Context, id int) (map[string]interface{}, error) {
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
