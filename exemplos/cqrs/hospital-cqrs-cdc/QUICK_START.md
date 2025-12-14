# 🚀 Quick Start - Hospital CQRS CDC

Guia rápido para executar o exemplo CQRS com CDC (Change Data Capture).

## ⚡ Início Rápido (3 minutos)

### 1️⃣ Subir Infraestrutura

```bash
cd hospital-cqrs-cdc
docker-compose up -d
```

Aguarde ~30 segundos para todos os serviços iniciarem.

### 2️⃣ Verificar Status

```bash
# Ver logs do Debezium Setup (deve criar connector automaticamente)
docker logs debezium-setup

# Deve mostrar:
# ✅ Connector created successfully
```

Verificar se connector está rodando:

```bash
curl -s http://localhost:8083/connectors/hospital-postgres-connector/status | jq
```

Saída esperada:
```json
{
  "name": "hospital-postgres-connector",
  "connector": {
    "state": "RUNNING"
  },
  "tasks": [{
    "state": "RUNNING"
  }]
}
```

### 3️⃣ Iniciar Command Service

```bash
cd cmd/command-service
go run main.go
```

Aguarde mensagem:
```
✅ Conectado ao PostgreSQL
🌐 Server rodando na porta 3001
```

### 4️⃣ Iniciar Event Handler (nova janela do terminal)

```bash
cd cmd/event-handler
go run main.go
```

Aguarde mensagem:
```
Event Handler aguardando eventos CDC do Debezium...
```

### 5️⃣ Testar o Fluxo

Criar uma prescrição:

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
      },
      {
        "id_medicamento": 2,
        "dosagem": "10mg",
        "horario": "12:00"
      }
    ]
  }'
```

Resposta esperada (201 Created):
```json
{
  "id": 1,
  "id_medico": 1,
  "id_paciente": 1,
  "data_prescricao": "2024-01-15T10:30:00Z"
}
```

### 6️⃣ Observar Eventos CDC

Nos logs do **Event Handler** você verá:

```
Evento CDC recebido do tópico hospital_db.public.prescricoes (offset: 0)
📋 Processando prescrição CDC: ID=1 Médico=1 Paciente=1
✅ View Farmácia atualizada para prescrição 1
✅ View Prontuário atualizada para prescrição 1
Evento CDC recebido do tópico hospital_db.public.prescricao_medicamentos (offset: 0)
💊 Processando medicamento CDC: Prescrição=1 Medicamento=1
✅ Medicamento CDC processado e views atualizadas
```

### 7️⃣ Consultar Views de Leitura

```bash
# Via PostgreSQL
docker exec -it postgres psql -U hospital -d hospital_db

# Consultar view da farmácia
SELECT * FROM view_farmacia;

# Consultar view do prontuário
SELECT * FROM view_prontuario_paciente;
```

---

## 🔍 Verificações Importantes

### ✅ Debezium está capturando?

```bash
# Listar tópicos Kafka
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list

# Você deve ver:
# hospital_db.public.prescricoes
# hospital_db.public.prescricao_medicamentos
```

Ver mensagens no tópico:

```bash
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic hospital_db.public.prescricoes \
  --from-beginning \
  --max-messages 1
```

### ✅ PostgreSQL com WAL habilitado?

```bash
docker exec postgres psql -U hospital -d hospital_db \
  -c "SHOW wal_level;"
```

Esperado: `logical`

### ✅ Replication Slot criado?

```bash
docker exec postgres psql -U hospital -d hospital_db \
  -c "SELECT slot_name, plugin, active FROM pg_replication_slots;"
```

Esperado:
```
    slot_name     |  plugin  | active
------------------+----------+--------
 hospital_slot    | pgoutput | t
```

---

## 🌐 UIs de Monitoramento

### Kafka UI
```bash
open http://localhost:9090
```

- Ver mensagens nos tópicos CDC
- Monitorar consumer groups
- Verificar lag

### Debezium UI
```bash
open http://localhost:8080
```

- Status do connector
- Configuração
- Métricas de captura

---

## 🧪 Casos de Teste

### Teste 1: Prescrição com Múltiplos Medicamentos

```bash
curl -X POST http://localhost:3001/api/prescricoes \
  -H "Content-Type: application/json" \
  -d '{
    "id_medico": 2,
    "id_paciente": 2,
    "medicamentos": [
      {"id_medicamento": 1, "dosagem": "500mg", "horario": "08:00"},
      {"id_medicamento": 2, "dosagem": "10mg", "horario": "12:00"},
      {"id_medicamento": 3, "dosagem": "200ml", "horario": "18:00"}
    ]
  }'
```

**Resultado Esperado**:
- 1 evento em `hospital_db.public.prescricoes`
- 3 eventos em `hospital_db.public.prescricao_medicamentos`
- Views atualizadas com todos os medicamentos

### Teste 2: Diferentes Médicos e Pacientes

```bash
# Médico Cardiologista para Paciente Ana
curl -X POST http://localhost:3001/api/prescricoes \
  -H "Content-Type: application/json" \
  -d '{
    "id_medico": 3,
    "id_paciente": 3,
    "medicamentos": [
      {"id_medicamento": 4, "dosagem": "100mg", "horario": "09:00"}
    ]
  }'
```

**Verificar**:
```sql
SELECT 
  medico_nome, 
  medico_especialidade,
  paciente_nome,
  medicamento_nome
FROM view_prontuario_paciente
WHERE id_prescricao = (SELECT MAX(id) FROM prescricoes);
```

### Teste 3: Alta Frequência (Stress Test)

```bash
# Criar múltiplas prescrições rapidamente
for i in {1..10}; do
  curl -X POST http://localhost:3001/api/prescricoes \
    -H "Content-Type: application/json" \
    -d "{
      \"id_medico\": $((1 + RANDOM % 3)),
      \"id_paciente\": $((1 + RANDOM % 5)),
      \"medicamentos\": [{
        \"id_medicamento\": $((1 + RANDOM % 5)),
        \"dosagem\": \"500mg\",
        \"horario\": \"08:00\"
      }]
    }" &
done
wait
```

**Verificar Lag**:
```bash
docker exec kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --group event-handler-cdc-group \
  --describe
```

---

## 🐛 Troubleshooting Rápido

### Problema: Connector não inicia

```bash
# Ver erro detalhado
docker logs kafka-connect

# Solução comum: Aguardar Kafka
# O debezium-setup já faz retry, mas você pode recriar:
docker restart debezium-setup
```

### Problema: Eventos não chegam

```bash
# 1. Verificar se comando foi para o banco
docker exec postgres psql -U hospital -d hospital_db \
  -c "SELECT COUNT(*) FROM prescricoes;"

# 2. Verificar se Debezium está lendo WAL
docker exec postgres psql -U hospital -d hospital_db \
  -c "SELECT * FROM pg_stat_replication;"

# 3. Verificar tópicos Kafka
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list
```

### Problema: Event Handler não processa

```bash
# Verificar se está consumindo
docker exec kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --group event-handler-cdc-group \
  --describe

# Ver mensagem de erro nos logs
# (checar terminal onde event-handler está rodando)
```

---

## 🛑 Parar Tudo

```bash
# Parar serviços Go (Ctrl+C nos terminais)

# Parar containers
docker-compose down

# Limpar volumes (apaga dados)
docker-compose down -v
```

---

## 📚 Próximos Passos

Agora que está funcionando, explore:

1. **README.md** - Documentação completa
2. **ARCHITECTURE_COMPARISON.md** - Comparação Manual vs CDC
3. **Código Fonte**:
   - `internal/events/cdc_handler.go` - Lógica de processamento CDC
   - `cmd/command-service/main.go` - API simplificada (sem Kafka)
   - `debezium/postgres-connector.json` - Configuração Debezium

---

## 🎯 Diferença Fundamental

**Versão Manual** (hospital-cqrs-sql):
```go
// Handler PUBLICA evento manualmente
kafka.Publish("prescricao.criada", event)
```

**Versão CDC** (hospital-cqrs-cdc):
```go
// Handler apenas PERSISTE
// Debezium captura do WAL automaticamente!
return c.Status(201).JSON(result)
```

---

## 🔗 Links Úteis

- Debezium UI: http://localhost:8080
- Kafka UI: http://localhost:9090
- Command API: http://localhost:3001
- Query API: http://localhost:3002 (se iniciada)
- Kafka Connect REST: http://localhost:8083

**Pronto! Você tem um CQRS com CDC funcionando! 🎉**
