# Hospital CQRS - Versão CDC (Change Data Capture)

Sistema de prescrições médicas demonstrando padrão CQRS com **Change Data Capture (CDC)** usando **Debezium**.

## 🎯 Diferença Fundamental

Ao contrário da versão `hospital-cqrs-sql` que publica eventos **manualmente** via código, esta versão utiliza **CDC** para capturar mudanças automaticamente do banco de dados.

### Comparação das Abordagens

| Aspecto | Manual (hospital-cqrs-sql) | CDC (hospital-cqrs-cdc) |
|---------|---------------------------|------------------------|
| **Emissão de Eventos** | Código da aplicação publica no Kafka | Debezium captura do Write-Ahead Log (WAL) |
| **Acoplamento** | Command Service depende do Kafka | Command Service independente do Kafka |
| **Complexidade** | Handler precisa publicar eventos | Handler só persiste no banco |
| **Garantias** | Eventual consistency manual | Atomic capture (mesma transação) |
| **Latência** | Menor (síncrono) | Ligeiramente maior (assíncrono) |
| **Resiliência** | Falha no Kafka = falha na API | Falha no Kafka não afeta escrita |

## 🏗️ Arquitetura CDC

```
┌─────────────────┐
│  Command API    │  ─────►  Escreve apenas no PostgreSQL
│ (Fiber + Go)    │           (sem dependência do Kafka)
└─────────────────┘
         │
         ▼
┌─────────────────┐
│   PostgreSQL    │
│  (WAL Enabled)  │
│                 │
│ ┌─────────────┐ │
│ │ Prescricoes │ │
│ └─────────────┘ │
└─────────────────┘
         │
         │ (WAL Replication)
         ▼
┌─────────────────┐
│   Debezium      │  ─────►  Transforma mudanças em eventos
│   Connector     │           e publica no Kafka
└─────────────────┘
         │
         ▼
┌─────────────────┐
│  Apache Kafka   │
│  Topics:        │
│  - hospital_db. │
│    public.      │
│    prescricoes  │
└─────────────────┘
         │
         ▼
┌─────────────────┐
│ Event Handler   │  ─────►  Atualiza Views de Leitura
│  (Consumer)     │
└─────────────────┘
         │
         ▼
┌─────────────────┐
│  Query Views    │
│  - Farmácia     │
│  - Prontuário   │
└─────────────────┘
```

## 🔧 Componentes

### 1. PostgreSQL com Logical Replication

```yaml
# Write-Ahead Log habilitado
wal_level: logical
max_wal_senders: 10
max_replication_slots: 10
```

### 2. Debezium Connect

- **Porta**: 8083
- **Plugin**: PostgreSQL Connector (pgoutput)
- **Transformações**: ExtractNewRecordState (unwrap)
- **Snapshot**: initial (captura dados existentes)

### 3. Command Service (Simplificado)

```go
// SEM dependência do Kafka!
func (h *PrescricaoHandler) CriarPrescricao(c *fiber.Ctx) error {
    // Apenas valida e persiste
    result, err := h.repo.CriarPrescricao(ctx, prescricao)
    
    // Debezium se encarrega do evento
    return c.Status(201).JSON(result)
}
```

### 4. Event Handler (CDC Consumer)

Consome eventos no formato Debezium:

```json
{
  "before": null,
  "after": {
    "id": 1,
    "id_medico": 5,
    "id_paciente": 10,
    "data_prescricao": "2024-01-15T10:30:00Z"
  },
  "source": {
    "version": "2.5.0.Final",
    "connector": "postgresql",
    "name": "hospital_db",
    "ts_ms": 1705315800000,
    "snapshot": "false",
    "db": "hospital_db",
    "schema": "public",
    "table": "prescricoes"
  },
  "op": "c",  // c=create, u=update, d=delete
  "ts_ms": 1705315800123
}
```

## 🚀 Como Executar

### Pré-requisitos

- Docker & Docker Compose
- Go 1.21+

### Passo 1: Subir Infraestrutura

```bash
cd hospital-cqrs-cdc
docker-compose up -d
```

Serviços iniciados:
- PostgreSQL (5432) - com WAL habilitado
- Kafka (9092) - KRaft mode
- Kafka UI (9090)
- Debezium Connect (8083)
- Debezium UI (8080)
- Debezium Setup (configura connector automaticamente)

### Passo 2: Verificar Debezium

Aguarde ~20 segundos e verifique se o connector foi criado:

```bash
# Via Debezium UI
open http://localhost:8080

# Via API
curl http://localhost:8083/connectors/hospital-postgres-connector/status
```

Resposta esperada:
```json
{
  "name": "hospital-postgres-connector",
  "connector": {
    "state": "RUNNING",
    "worker_id": "kafka-connect:8083"
  },
  "tasks": [{
    "id": 0,
    "state": "RUNNING",
    "worker_id": "kafka-connect:8083"
  }]
}
```

### Passo 3: Verificar Tópicos Criados

```bash
# Lista tópicos
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list

# Esperado:
# hospital_db.public.prescricoes
# hospital_db.public.prescricao_medicamentos
```

### Passo 4: Iniciar Command Service

```bash
cd cmd/command-service
go run main.go
```

Logs esperados:
```
🚀 Iniciando Command Service (CDC Mode - sem Kafka Producer)...
✅ Conectado ao PostgreSQL
🌐 Server rodando na porta 3001
```

### Passo 5: Iniciar Event Handler

```bash
cd cmd/event-handler
go run main.go
```

Logs esperados:
```
🚀 Iniciando Event Handler Service (CDC Mode)...
✅ Conectado ao PostgreSQL
✅ Conectado ao Kafka
📡 Consumindo tópicos CDC: [hospital_db.public.prescricoes hospital_db.public.prescricao_medicamentos]
Event Handler aguardando eventos CDC do Debezium...
```

### Passo 6: Criar Prescrição

```bash
curl -X POST http://localhost:3001/api/prescricoes \
  -H "Content-Type: application/json" \
  -d '{
    "id_medico": 1,
    "id_paciente": 1,
    "medicamentos": [
      {
        "id_medicamento": 1,
        "dosagem": "500mg",
        "horario": "08:00"
      }
    ]
  }'
```

### Passo 7: Observar Fluxo CDC

#### No PostgreSQL (escrita)
```sql
SELECT * FROM prescricoes ORDER BY id DESC LIMIT 1;
```

#### No Kafka (eventos CDC capturados)
```bash
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic hospital_db.public.prescricoes \
  --from-beginning
```

#### Nos Logs do Event Handler
```
📨 Evento CDC recebido do tópico hospital_db.public.prescricoes (offset: 0)
📋 Processando prescrição CDC: ID=1 Médico=1 Paciente=1
✅ View Farmácia atualizada para prescrição 1
✅ View Prontuário atualizada para prescrição 1
✅ Evento CDC processado: Views atualizadas para prescrição 1
```

#### Nas Views (leitura)
```sql
SELECT * FROM view_farmacia;
SELECT * FROM view_prontuario_paciente;
```

## 🔍 Monitoramento

### Debezium UI
```bash
open http://localhost:8080
```

- Status do connector
- Métricas de captura
- Lag de replicação

### Kafka UI
```bash
open http://localhost:9090
```

- Mensagens nos tópicos
- Consumer groups
- Offsets

### PostgreSQL Replication Slots
```sql
SELECT * FROM pg_replication_slots;
```

Mostra o slot usado pelo Debezium para rastrear posição no WAL.

## 🎯 Vantagens do CDC

### 1. **Desacoplamento**
Command Service não precisa conhecer Kafka. Simplifica código e testes.

### 2. **Garantias Atômicas**
Eventos são capturados da mesma transação do banco. Sem dual-write problem.

### 3. **Captura Completa**
Captura INSERT, UPDATE, DELETE automaticamente (não apenas eventos de negócio).

### 4. **Resiliência**
Se Kafka cair, aplicação continua funcionando. Debezium replica quando volta.

### 5. **Auditoria Natural**
WAL já existe para durabilidade. CDC aproveita infraestrutura existente.

### 6. **Histórico Completo**
Snapshot inicial + mudanças incrementais = histórico completo.

## ⚠️ Desvantagens do CDC

### 1. **Latência Adicional**
Eventos não são síncronos com a transação (ms a segundos de delay).

### 2. **Eventos Técnicos**
Captura mudanças de banco, não eventos de negócio semânticos.

### 3. **Schema Coupling**
Mudanças no schema do banco afetam formato dos eventos.

### 4. **Complexidade Operacional**
Mais componentes para monitorar (Debezium, replication slots, WAL).

### 5. **Débito Técnico**
Dificulta migração de banco ou mudança de estrutura.

## 📊 Quando Usar CDC?

### ✅ Use CDC Quando:

- Command side precisa de **máxima simplicidade**
- Você quer **garantias transacionais** fortes
- Já usa PostgreSQL com WAL
- Precisa capturar **todas** as mudanças (inclusive fora da app)
- Latência de alguns segundos é aceitável

### ❌ Evite CDC Quando:

- Precisa de **eventos semânticos de negócio**
- Latência deve ser **mínima** (< 100ms)
- Schema do banco muda frequentemente
- Quer eventos ricos com contexto de negócio
- Equipe não tem expertise em Debezium/CDC

## 🔧 Configurações Importantes

### Debezium Connector

```json
{
  "plugin.name": "pgoutput",  // Plugin nativo do PostgreSQL 10+
  "slot.name": "hospital_slot",  // Replication slot
  "publication.name": "hospital_publication",  // Publicação lógica
  "snapshot.mode": "initial",  // Captura dados existentes no início
  "transforms": "unwrap",  // Extrai payload do envelope
  "transforms.unwrap.type": "io.debezium.transforms.ExtractNewRecordState"
}
```

### PostgreSQL

```conf
# postgresql.conf
wal_level = logical  # Habilita replicação lógica
max_wal_senders = 10  # Máximo de processos de envio WAL
max_replication_slots = 10  # Máximo de slots de replicação
```

## 🐛 Troubleshooting

### Connector Não Inicia

```bash
# Verificar logs do Debezium
docker logs kafka-connect

# Erro comum: WAL não habilitado
# Solução: Verificar wal_level no PostgreSQL
docker exec postgres psql -U hospital -d hospital_db -c "SHOW wal_level;"
```

### Eventos Não Chegam

```bash
# Verificar replication slot
docker exec postgres psql -U hospital -d hospital_db \
  -c "SELECT * FROM pg_replication_slots;"

# Verificar se slot está ativo
# Se confirmed_flush_lsn não muda, há problema
```

### Consumer Lag Crescendo

```bash
# Ver offset do consumer
docker exec kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --group event-handler-cdc-group \
  --describe

# Escalar event-handler se necessário
```

## 📚 Referências

- [Debezium PostgreSQL Connector](https://debezium.io/documentation/reference/stable/connectors/postgresql.html)
- [PostgreSQL Logical Replication](https://www.postgresql.org/docs/current/logical-replication.html)
- [Change Data Capture Pattern](https://microservices.io/patterns/data/transaction-log-tailing.html)

## 🔗 Comparação com Versão Manual

Para ver a implementação **sem CDC** (com publicação manual de eventos), veja:
```
../hospital-cqrs-sql/
```

Ambas implementações demonstram CQRS, mas com estratégias diferentes de event sourcing.
