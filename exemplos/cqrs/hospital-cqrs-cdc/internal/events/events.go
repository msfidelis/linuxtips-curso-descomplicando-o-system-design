package events

import "time"

// =========================================
// DOMAIN EVENTS
// =========================================

// EventType representa o tipo de evento
type EventType string

const (
	// PrescricaoCriadaEvent é disparado quando uma prescrição é criada
	PrescricaoCriadaEvent EventType = "prescricao.criada"
)

// Event representa um evento do domínio
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// =========================================
// PRESCRICAO CRIADA EVENT
// =========================================

// MedicamentoPrescritoEvent representa um medicamento em um evento
type MedicamentoPrescritoEvent struct {
	IDMedicamento int    `json:"id_medicamento"`
	Horario       string `json:"horario"`
	Dosagem       string `json:"dosagem"`
}

// PrescricaoCriadaEventData contém os dados do evento de prescrição criada
type PrescricaoCriadaEventData struct {
	IDPrescricao   int                         `json:"id_prescricao"`
	IDMedico       int                         `json:"id_medico"`
	IDPaciente     int                         `json:"id_paciente"`
	DataPrescricao time.Time                   `json:"data_prescricao"`
	Medicamentos   []MedicamentoPrescritoEvent `json:"medicamentos"`
}

// NewPrescricaoCriadaEvent cria um novo evento de prescrição criada
func NewPrescricaoCriadaEvent(data PrescricaoCriadaEventData) Event {
	return Event{
		ID:        generateEventID(),
		Type:      PrescricaoCriadaEvent,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"id_prescricao":   data.IDPrescricao,
			"id_medico":       data.IDMedico,
			"id_paciente":     data.IDPaciente,
			"data_prescricao": data.DataPrescricao,
			"medicamentos":    data.Medicamentos,
		},
	}
}

// generateEventID gera um ID único para o evento
func generateEventID() string {
	return time.Now().Format("20060102150405.000000")
}
