# Arquitetura CQRS - Comparação Manual vs CDC

## 🔀 Visão Geral das Duas Abordagens

Este documento compara as duas implementações de CQRS fornecidas neste repositório.

---

## 📐 Arquitetura Manual (hospital-cqrs-sql)

```
┌──────────────────────────────────────────────────────────────┐
│                    COMMAND SIDE (Write)                       │
└──────────────────────────────────────────────────────────────┘
                             │
                             ▼
        ┌─────────────────────────────────────┐
        │       Command API (Fiber)           │
        │  POST /api/prescricoes              │
        └─────────────────────────────────────┘
                             │
                             ▼
        ┌─────────────────────────────────────┐
        │    PrescricaoHandler.go             │
        │  1. Valida dados                    │
        │  2. Persiste no PostgreSQL ─────────┼──► PostgreSQL
        │  3. Publica evento no Kafka ────────┼──► Kafka Producer
        └─────────────────────────────────────┘
                             │
                             │ (Dual Write)
                             ▼
              ┌──────────────────────────┐
              │    Apache Kafka          │
              │  Topic: prescricao.criada│
              └──────────────────────────┘
                             │
                             ▼
        ┌─────────────────────────────────────┐
        │      Event Handler (Consumer)       │
        │  1. Consome evento                  │
        │  2. Busca dados completos           │
        │  3. Atualiza View_Farmacia          │
        │  4. Atualiza View_Prontuario        │
        └─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────┐
│                      QUERY SIDE (Read)                        │
│                                                               │
│  ┌──────────────────┐       ┌──────────────────────┐        │
│  │  View_Farmacia   │       │ View_Prontuario      │        │
│  │  (Desnormalizada)│       │ (Desnormalizada)     │        │
│  └──────────────────┘       └──────────────────────┘        │
└──────────────────────────────────────────────────────────────┘
```

### Fluxo Detalhado (Manual)

1. **Client** → POST para Command API
2. **Command Handler**:
   - Inicia transação no PostgreSQL
   - INSERT em `Prescricoes`
   - INSERT em `Prescricao_Medicamentos`
   - COMMIT transação
   - **Publica evento no Kafka** (fora da transação!)
3. **Kafka** armazena evento
4. **Event Handler** consome evento
5. **Event Handler** atualiza views de leitura

#### ⚠️ Problema: Dual Write

```
┌─────────────────┐
│ PostgreSQL TX   │ ✅ COMMIT
└─────────────────┘
         │
         ▼
┌─────────────────┐
│ Kafka Producer  │ ❌ FALHA (rede, timeout, etc)
└─────────────────┘
```

**Resultado**: Dado persistido mas evento não publicado = **inconsistência**!

---

## 📐 Arquitetura CDC (hospital-cqrs-cdc)

```
┌──────────────────────────────────────────────────────────────┐
│                    COMMAND SIDE (Write)                       │
└──────────────────────────────────────────────────────────────┘
                             │
                             ▼
        ┌─────────────────────────────────────┐
        │       Command API (Fiber)           │
        │  POST /api/prescricoes              │
        └─────────────────────────────────────┘
                             │
                             ▼
        ┌─────────────────────────────────────┐
        │    PrescricaoHandler.go             │
        │  1. Valida dados                    │
        │  2. Persiste no PostgreSQL          │
        │  (NÃO publica eventos!)             │
        └─────────────────────────────────────┘
                             │
                             ▼
              ┌──────────────────────────┐
              │     PostgreSQL           │
              │  (wal_level = logical)   │
              │                          │
              │  Write-Ahead Log (WAL)   │
              └──────────────────────────┘
                             │
                             │ (Replicação Lógica)
                             ▼
        ┌─────────────────────────────────────┐
        │      Debezium Connector             │
        │  1. Lê WAL via replication slot     │
        │  2. Transforma mudanças em eventos  │
        │  3. Publica no Kafka                │
        └─────────────────────────────────────┘
                             │
                             ▼
              ┌──────────────────────────┐
              │    Apache Kafka          │
              │  hospital_db.public.     │
              │    prescricoes           │
              └──────────────────────────┘
                             │
                             ▼
        ┌─────────────────────────────────────┐
        │      Event Handler (Consumer)       │
        │  1. Consome evento CDC              │
        │  2. Parse formato Debezium          │
        │  3. Busca dados completos           │
        │  4. Atualiza View_Farmacia          │
        │  5. Atualiza View_Prontuario        │
        └─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────┐
│                      QUERY SIDE (Read)                        │
│                                                               │
│  ┌──────────────────┐       ┌──────────────────────┐        │
│  │  View_Farmacia   │       │ View_Prontuario      │        │
│  │  (Desnormalizada)│       │ (Desnormalizada)     │        │
│  └──────────────────┘       └──────────────────────┘        │
└──────────────────────────────────────────────────────────────┘
```

### Fluxo Detalhado (CDC)

1. **Client** → POST para Command API
2. **Command Handler**:
   - Inicia transação no PostgreSQL
   - INSERT em `Prescricoes`
   - INSERT em `Prescricao_Medicamentos`
   - COMMIT transação
   - **FIM** (não publica nada!)
3. **PostgreSQL** registra mudança no WAL
4. **Debezium** lê WAL via replication slot
5. **Debezium** publica eventos CDC no Kafka
6. **Event Handler** consome eventos
7. **Event Handler** atualiza views de leitura

#### ✅ Vantagem: Atomic Capture

```
┌─────────────────────────────────────────┐
│ PostgreSQL TX + WAL (mesma transação)   │ ✅ COMMIT
└─────────────────────────────────────────┘
                  │
                  ▼
      ┌───────────────────────┐
      │ Debezium (assíncrono) │ ✅ Garante entrega eventual
      └───────────────────────┘
```

**Resultado**: Se commit sucedeu, WAL tem o registro. Debezium **garante** publicação eventual.

---

## 📊 Comparação Detalhada

| Característica | Manual (hospital-cqrs-sql) | CDC (hospital-cqrs-cdc) |
|---------------|---------------------------|------------------------|
| **Acoplamento com Kafka** | ❌ Alto - Handler precisa de Kafka Producer | ✅ Baixo - Handler independente |
| **Complexidade do Code** | ❌ Médio - Dual write manual | ✅ Simples - Apenas persiste |
| **Garantias de Entrega** | ⚠️ Best-effort (pode falhar) | ✅ At-least-once (garantido pelo WAL) |
| **Consistência** | ⚠️ Eventual (com risco de falha) | ✅ Eventual (sem risco) |
| **Latência de Evento** | ✅ Baixa (~ms) | ⚠️ Média (~segundos) |
| **Testabilidade** | ❌ Precisa mockar Kafka | ✅ Testa só persistência |
| **Resiliência** | ❌ Kafka down = API down | ✅ Kafka down = API funciona |
| **Overhead Operacional** | ✅ Baixo | ❌ Médio (Debezium, slots, WAL) |
| **Tipo de Evento** | ✅ Semântico (negócio) | ⚠️ Técnico (mudança de DB) |
| **Schema Evolution** | ✅ Flexível | ❌ Acoplado ao DB schema |
| **Captura de Mudanças** | ⚠️ Apenas eventos explícitos | ✅ Todas mudanças (INSERT/UPDATE/DELETE) |
| **Debugging** | ✅ Direto (logs da app) | ⚠️ Indireto (logs Debezium + app) |

---

## 🎯 Quando Usar Cada Abordagem?

### Use **Manual** (hospital-cqrs-sql) quando:

- ✅ Precisa de **eventos semânticos de negócio**
- ✅ Latência deve ser **mínima** (tempo real)
- ✅ Eventos precisam ter **contexto de negócio rico**
- ✅ Schema do banco muda com frequência
- ✅ Equipe não tem expertise em CDC
- ✅ Quer **controle total** sobre o que vira evento
- ✅ Precisa de eventos **idempotentes** bem definidos

**Exemplo Real**: E-commerce onde "Pedido Criado" precisa ter contexto completo (itens, promoções, cupons) imediatamente.

### Use **CDC** (hospital-cqrs-cdc) quando:

- ✅ Command side precisa de **máxima simplicidade**
- ✅ Quer **garantias transacionais** fortes (sem dual-write)
- ✅ Já usa PostgreSQL e WAL está disponível
- ✅ Precisa capturar **todas** as mudanças (inclusive manuais via SQL)
- ✅ Latência de alguns segundos é aceitável
- ✅ Prefere **infraestrutura** resolver o problema
- ✅ Quer **auditoria completa** automática

**Exemplo Real**: Sistema bancário onde toda mudança no saldo precisa ser auditada, inclusive ajustes manuais de DBA.

---

## 🔧 Formato dos Eventos

### Manual - Evento Semântico

```json
{
  "tipo": "PRESCRICAO_CRIADA",
  "timestamp": "2024-01-15T10:30:00Z",
  "dados": {
    "id_prescricao": 123,
    "medico": {
      "id": 5,
      "nome": "Dr. João Silva",
      "crm": "12345-SP"
    },
    "paciente": {
      "id": 10,
      "nome": "Maria Santos"
    },
    "medicamentos": [
      {
        "nome": "Paracetamol",
        "dosagem": "500mg",
        "horario": "08:00"
      }
    ]
  }
}
```

**Vantagens**:
- Rico em contexto de negócio
- Fácil de entender e consumir
- Versionável (pode evoluir independente do DB)

### CDC - Evento Técnico (Debezium)

```json
{
  "before": null,
  "after": {
    "id": 123,
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
    "table": "prescricoes",
    "lsn": 33007024
  },
  "op": "c",  // create
  "ts_ms": 1705315800123
}
```

**Vantagens**:
- Captura mudanças automaticamente
- Inclui metadados completos (LSN, timestamp, snapshot)
- Suporta UPDATE e DELETE nativamente

**Desvantagens**:
- Não tem contexto de negócio
- Precisa joins para enriquecer
- Acoplado ao schema do banco

---

## 🏗️ Infraestrutura

### Manual

```yaml
services:
  - postgres
  - kafka
  - command-service (precisa de Kafka client)
  - event-handler
  - query-service
```

**Total: 5 containers**

### CDC

```yaml
services:
  - postgres (com WAL habilitado)
  - kafka
  - kafka-connect (Debezium)
  - debezium-ui (opcional, monitoring)
  - command-service (SEM Kafka client)
  - event-handler
  - query-service
```

**Total: 6-7 containers** (mais Debezium)

---

## 🐛 Problemas Comuns

### Manual - Dual Write Problem

```go
// ❌ PROBLEMA: Não é atômico!
db.Commit()  // Sucesso
kafka.Publish(event)  // FALHA - rede caiu

// Resultado: Dado no banco, sem evento no Kafka
```

**Soluções**:
- Outbox Pattern (tabela de eventos)
- Two-Phase Commit (complexo)
- Aceitar inconsistência eventual

### CDC - Latência

```
Write: t=0s
WAL: t=0.1s
Debezium lê: t=2s
Kafka: t=2.5s
Consumer: t=3s
View atualizada: t=3.5s
```

**Mitigações**:
- Tuning do Debezium (poll interval)
- Cache nas views
- Aceitar eventual consistency

---

## 🎓 Casos de Uso Ideais

### Manual (hospital-cqrs-sql)

1. **E-commerce - Carrinho de Compras**
   - Eventos: `ItemAdicionado`, `CarrinhoFinalizado`
   - Precisa: Contexto rico, baixa latência

2. **Sistema de Notificações**
   - Eventos: `UsuarioCriado`, `SenhaAlterada`
   - Precisa: Controle fino sobre o que notificar

3. **Gaming - Matchmaking**
   - Eventos: `PartidaIniciada`, `JogadorConectou`
   - Precisa: Tempo real, eventos complexos

### CDC (hospital-cqrs-cdc)

1. **Banking - Auditoria de Transações**
   - Captura: Toda mudança em contas
   - Precisa: Garantias fortes, auditoria completa

2. **Data Warehouse ETL**
   - Captura: Mudanças em tabelas operacionais
   - Precisa: Replicação confiável

3. **Compliance / LGPD**
   - Captura: Alterações em dados sensíveis
   - Precisa: Log imutável de mudanças

---

## 🔄 Evolução e Migração

### De Manual para CDC

1. Manter código atual funcionando
2. Adicionar Debezium em paralelo
3. Comparar eventos (manual vs CDC)
4. Gradualmente remover publicação manual
5. Simplificar handlers

### De CDC para Manual

1. Implementar event publishers
2. Adicionar lógica de negócio nos eventos
3. Testar em paralelo
4. Remover Debezium
5. Desabilitar WAL (se não usado para outros fins)

---

## 📖 Conclusão

Ambas as abordagens são válidas e resolvem CQRS de formas diferentes:

- **Manual**: Mais controle, mais código, eventos ricos
- **CDC**: Menos código, mais infraestrutura, eventos técnicos

A escolha depende de:
- Requisitos de latência
- Complexidade aceitável
- Expertise da equipe
- Necessidades de auditoria
- Garantias de consistência

**Recomendação Geral**:
- Comece com **Manual** se sua equipe é pequena e precisa de simplicidade
- Use **CDC** se precisa de garantias fortes e tem ops para gerenciar Debezium
- Considere **Outbox Pattern** como meio-termo (melhor dos dois mundos)
