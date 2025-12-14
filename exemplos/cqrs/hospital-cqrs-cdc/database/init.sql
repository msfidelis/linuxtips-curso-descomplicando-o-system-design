-- =========================================
-- HOSPITAL CQRS - DATABASE INITIALIZATION
-- =========================================

-- =========================================
-- COMMAND MODEL (Write Side - Normalized)
-- =========================================

-- Tabela de Médicos
CREATE TABLE IF NOT EXISTS Medicos (
    id SERIAL PRIMARY KEY,
    nome VARCHAR(255) NOT NULL,
    especialidade VARCHAR(255) NOT NULL,
    crm VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de Pacientes
CREATE TABLE IF NOT EXISTS Pacientes (
    id SERIAL PRIMARY KEY,
    nome VARCHAR(255) NOT NULL,
    data_nascimento DATE NOT NULL,
    endereco VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de Medicamentos
CREATE TABLE IF NOT EXISTS Medicamentos (
    id SERIAL PRIMARY KEY,
    nome VARCHAR(255) NOT NULL,
    descricao TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de Prescrições (Modelo de Comando)
CREATE TABLE IF NOT EXISTS Prescricoes (
    id SERIAL PRIMARY KEY,
    id_medico INT NOT NULL,
    id_paciente INT NOT NULL,
    data_prescricao TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id_medico) REFERENCES Medicos(id),
    FOREIGN KEY (id_paciente) REFERENCES Pacientes(id)
);

-- Tabela de Relacionamento Prescrição-Medicamentos
CREATE TABLE IF NOT EXISTS Prescricao_Medicamentos (
    id SERIAL PRIMARY KEY,
    id_prescricao INT NOT NULL,
    id_medicamento INT NOT NULL,
    horario VARCHAR(50) NOT NULL,
    dosagem VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id_prescricao) REFERENCES Prescricoes(id) ON DELETE CASCADE,
    FOREIGN KEY (id_medicamento) REFERENCES Medicamentos(id)
);

-- Índices para otimizar buscas no modelo de comando
CREATE INDEX idx_prescricoes_medico ON Prescricoes(id_medico);
CREATE INDEX idx_prescricoes_paciente ON Prescricoes(id_paciente);
CREATE INDEX idx_prescricao_medicamentos_prescricao ON Prescricao_Medicamentos(id_prescricao);

-- =========================================
-- QUERY MODELS (Read Side - Denormalized)
-- =========================================

-- Query Model 1: View de Farmácia
-- Modelo otimizado para a farmácia visualizar prescrições e separar medicamentos
CREATE TABLE IF NOT EXISTS View_Farmacia (
    id SERIAL PRIMARY KEY,
    id_prescricao INT NOT NULL,
    data_prescricao TIMESTAMP NOT NULL,
    
    -- Dados do Paciente
    paciente_id INT NOT NULL,
    paciente_nome VARCHAR(255) NOT NULL,
    paciente_data_nascimento DATE NOT NULL,
    
    -- Dados do Medicamento
    medicamento_id INT NOT NULL,
    medicamento_nome VARCHAR(255) NOT NULL,
    medicamento_descricao TEXT,
    horario VARCHAR(50) NOT NULL,
    dosagem VARCHAR(50) NOT NULL,
    
    -- Metadados
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índices para otimizar consultas da farmácia
CREATE INDEX idx_view_farmacia_prescricao ON View_Farmacia(id_prescricao);
CREATE INDEX idx_view_farmacia_paciente ON View_Farmacia(paciente_id);
CREATE INDEX idx_view_farmacia_medicamento ON View_Farmacia(medicamento_id);
CREATE INDEX idx_view_farmacia_data ON View_Farmacia(data_prescricao);

-- Query Model 2: View de Prontuário do Paciente
-- Modelo otimizado para visualizar histórico completo do paciente
CREATE TABLE IF NOT EXISTS View_Prontuario_Paciente (
    id SERIAL PRIMARY KEY,
    id_prescricao INT NOT NULL,
    data_prescricao TIMESTAMP NOT NULL,
    
    -- Dados do Paciente
    paciente_id INT NOT NULL,
    paciente_nome VARCHAR(255) NOT NULL,
    paciente_data_nascimento DATE NOT NULL,
    paciente_endereco VARCHAR(255),
    
    -- Dados do Médico
    medico_id INT NOT NULL,
    medico_nome VARCHAR(255) NOT NULL,
    medico_especialidade VARCHAR(255) NOT NULL,
    medico_crm VARCHAR(255) NOT NULL,
    
    -- Dados do Medicamento
    medicamento_id INT NOT NULL,
    medicamento_nome VARCHAR(255) NOT NULL,
    medicamento_descricao TEXT,
    horario VARCHAR(50) NOT NULL,
    dosagem VARCHAR(50) NOT NULL,
    
    -- Metadados
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índices para otimizar consultas de prontuário
CREATE INDEX idx_view_prontuario_prescricao ON View_Prontuario_Paciente(id_prescricao);
CREATE INDEX idx_view_prontuario_paciente ON View_Prontuario_Paciente(paciente_id);
CREATE INDEX idx_view_prontuario_medico ON View_Prontuario_Paciente(medico_id);
CREATE INDEX idx_view_prontuario_data ON View_Prontuario_Paciente(data_prescricao);

-- =========================================
-- DADOS DE EXEMPLO (SEED)
-- =========================================

-- Inserir médicos de exemplo
INSERT INTO Medicos (nome, especialidade, crm) VALUES
('Dr. João Silva', 'Cardiologia', 'CRM-SP-123456'),
('Dra. Maria Santos', 'Clínica Geral', 'CRM-RJ-789012'),
('Dr. Pedro Oliveira', 'Pediatria', 'CRM-MG-345678'),
('Dra. Ana Costa', 'Dermatologia', 'CRM-SP-901234'),
('Dr. Carlos Mendes', 'Ortopedia', 'CRM-RS-567890')
ON CONFLICT (crm) DO NOTHING;

-- Inserir pacientes de exemplo
INSERT INTO Pacientes (nome, data_nascimento, endereco) VALUES
('José da Silva', '1980-05-15', 'Rua das Flores, 123 - São Paulo, SP'),
('Maria Oliveira', '1992-08-22', 'Av. Paulista, 1000 - São Paulo, SP'),
('João Santos', '1975-03-10', 'Rua XV de Novembro, 456 - Rio de Janeiro, RJ'),
('Ana Paula', '2010-12-05', 'Rua das Acácias, 789 - Belo Horizonte, MG'),
('Carlos Eduardo', '1988-07-18', 'Av. Ipiranga, 321 - Porto Alegre, RS'),
('Fernanda Lima', '1995-11-30', 'Rua Aurora, 555 - São Paulo, SP'),
('Roberto Costa', '1968-09-25', 'Rua da Consolação, 888 - São Paulo, SP');

-- Inserir medicamentos de exemplo
INSERT INTO Medicamentos (nome, descricao) VALUES
('Paracetamol 500mg', 'Analgésico e antipirético'),
('Amoxicilina 500mg', 'Antibiótico de amplo espectro'),
('Omeprazol 20mg', 'Inibidor de bomba de prótons para tratamento de úlcera'),
('Dipirona 500mg', 'Analgésico e antitérmico'),
('Ibuprofeno 600mg', 'Anti-inflamatório não esteroidal'),
('Losartana 50mg', 'Anti-hipertensivo'),
('Metformina 850mg', 'Antidiabético oral'),
('Sinvastatina 20mg', 'Redutor de colesterol'),
('Enalapril 10mg', 'Anti-hipertensivo inibidor da ECA'),
('Captopril 25mg', 'Anti-hipertensivo');
