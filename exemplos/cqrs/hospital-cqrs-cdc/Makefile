# Makefile - Hospital CQRS CDC
# Comandos úteis para gerenciar o projeto

.PHONY: help up down logs restart clean test dev-command dev-event dev-query status kafka-ui debezium-ui

# Cores para output
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
BLUE   := $(shell tput -Txterm setaf 4)
RESET  := $(shell tput -Txterm sgr0)

help: ## Mostra este menu de ajuda
	@echo "${BLUE}Hospital CQRS - CDC Mode${RESET}"
	@echo ""
	@echo "${GREEN}Comandos disponíveis:${RESET}"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  ${YELLOW}%-20s${RESET} %s\n", $$1, $$2}'

up: ## Sobe toda a infraestrutura (PostgreSQL, Kafka, Debezium)
	@echo "${GREEN}🚀 Iniciando infraestrutura...${RESET}"
	docker-compose up -d
	@echo "${GREEN}✅ Aguarde ~30s para Debezium configurar o connector${RESET}"

down: ## Para todos os containers
	@echo "${YELLOW}🛑 Parando containers...${RESET}"
	docker-compose down

clean: ## Remove containers e volumes (apaga dados)
	@echo "${YELLOW}🧹 Limpando tudo...${RESET}"
	docker-compose down -v
	rm -rf tmp/

restart: down up ## Reinicia toda a infraestrutura

logs: ## Mostra logs de todos os containers
	docker-compose logs -f

logs-debezium: ## Mostra logs do Debezium Connect
	docker logs -f kafka-connect

logs-postgres: ## Mostra logs do PostgreSQL
	docker logs -f postgres

logs-kafka: ## Mostra logs do Kafka
	docker logs -f kafka

status: ## Verifica status de todos os serviços
	@echo "${BLUE}📊 Status dos Serviços${RESET}"
	@echo ""
	@echo "${YELLOW}Docker Containers:${RESET}"
	@docker-compose ps
	@echo ""
	@echo "${YELLOW}Debezium Connector:${RESET}"
	@curl -s http://localhost:8083/connectors/hospital-postgres-connector/status 2>/dev/null | \
		jq -r '"  Estado: " + .connector.state' || echo "  ❌ Não disponível"
	@echo ""
	@echo "${YELLOW}PostgreSQL WAL:${RESET}"
	@docker exec postgres psql -U hospital -d hospital_db \
		-t -c "SELECT '  wal_level: ' || setting FROM pg_settings WHERE name = 'wal_level';" 2>/dev/null || echo "  ❌ Não disponível"
	@echo ""
	@echo "${YELLOW}Replication Slots:${RESET}"
	@docker exec postgres psql -U hospital -d hospital_db \
		-t -c "SELECT '  ' || slot_name || ' (active: ' || active || ')' FROM pg_replication_slots;" 2>/dev/null || echo "  ❌ Não disponível"

kafka-topics: ## Lista tópicos Kafka
	@echo "${BLUE}📡 Tópicos Kafka (CDC)${RESET}"
	@docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-ui: ## Abre Kafka UI no browser
	@echo "${GREEN}🌐 Abrindo Kafka UI...${RESET}"
	@open http://localhost:9090 || xdg-open http://localhost:9090

debezium-ui: ## Abre Debezium UI no browser
	@echo "${GREEN}🌐 Abrindo Debezium UI...${RESET}"
	@open http://localhost:8080 || xdg-open http://localhost:8080

dev-command: ## Inicia Command Service em modo dev (Air hot reload)
	@echo "${GREEN}🔥 Iniciando Command Service (Hot Reload)${RESET}"
	air -c .air.command.toml

dev-event: ## Inicia Event Handler em modo dev (Air hot reload)
	@echo "${GREEN}🔥 Iniciando Event Handler (Hot Reload)${RESET}"
	air -c .air.event.toml

dev-query: ## Inicia Query Service em modo dev (Air hot reload)
	@echo "${GREEN}🔥 Iniciando Query Service (Hot Reload)${RESET}"
	air -c .air.query.toml

test-create: ## Cria uma prescrição de teste
	@echo "${BLUE}📝 Criando prescrição de teste...${RESET}"
	@curl -X POST http://localhost:3001/api/prescricoes \
		-H "Content-Type: application/json" \
		-d '{"id_medico":1,"id_paciente":1,"medicamentos":[{"id_medicamento":1,"dosagem":"500mg","horario":"08:00"}]}' \
		| jq '.'

test-query: ## Consulta views de leitura
	@echo "${BLUE}🔍 Consultando Views...${RESET}"
	@echo ""
	@echo "${YELLOW}View Farmácia:${RESET}"
	@docker exec postgres psql -U hospital -d hospital_db \
		-c "SELECT id_prescricao, paciente_nome, medicamento_nome, dosagem, horario FROM view_farmacia LIMIT 5;"
	@echo ""
	@echo "${YELLOW}View Prontuário:${RESET}"
	@docker exec postgres psql -U hospital -d hospital_db \
		-c "SELECT id_prescricao, medico_nome, paciente_nome, medicamento_nome FROM view_prontuario_paciente LIMIT 5;"

psql: ## Conecta ao PostgreSQL via psql
	@docker exec -it postgres psql -U hospital -d hospital_db

kafka-console: ## Consome mensagens de um tópico (ex: make kafka-console TOPIC=hospital_db.public.prescricoes)
	@docker exec kafka kafka-console-consumer \
		--bootstrap-server localhost:9092 \
		--topic $(TOPIC) \
		--from-beginning

watch-events: ## Monitora eventos CDC em tempo real
	@echo "${BLUE}👀 Monitorando eventos CDC (Ctrl+C para sair)${RESET}"
	@docker exec kafka kafka-console-consumer \
		--bootstrap-server localhost:9092 \
		--topic hospital_db.public.prescricoes \
		--from-beginning | jq '.'

setup-connector: ## (Re)cria o connector Debezium
	@echo "${YELLOW}🔧 Recriando connector Debezium...${RESET}"
	@curl -s -X DELETE http://localhost:8083/connectors/hospital-postgres-connector 2>/dev/null || true
	@sleep 2
	@curl -X POST http://localhost:8083/connectors \
		-H "Content-Type: application/json" \
		-d @debezium/postgres-connector.json
	@echo ""
	@echo "${GREEN}✅ Connector criado${RESET}"

verify-cdc: ## Verifica se CDC está funcionando
	@echo "${BLUE}🔍 Verificando CDC Pipeline${RESET}"
	@echo ""
	@echo "${YELLOW}1. PostgreSQL WAL:${RESET}"
	@docker exec postgres psql -U hospital -d hospital_db -c "SHOW wal_level;" | grep logical && echo "  ✅ WAL habilitado" || echo "  ❌ WAL não habilitado"
	@echo ""
	@echo "${YELLOW}2. Replication Slot:${RESET}"
	@docker exec postgres psql -U hospital -d hospital_db -t -c "SELECT COUNT(*) FROM pg_replication_slots WHERE slot_name = 'hospital_slot';" | grep -q 1 && echo "  ✅ Slot criado" || echo "  ❌ Slot não existe"
	@echo ""
	@echo "${YELLOW}3. Debezium Connector:${RESET}"
	@curl -s http://localhost:8083/connectors/hospital-postgres-connector/status | jq -r .connector.state | grep -q RUNNING && echo "  ✅ Connector rodando" || echo "  ❌ Connector com problema"
	@echo ""
	@echo "${YELLOW}4. Tópicos Kafka:${RESET}"
	@docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list | grep -q hospital_db && echo "  ✅ Tópicos CDC criados" || echo "  ❌ Tópicos não existem"

stress-test: ## Cria 20 prescrições rapidamente (teste de carga)
	@echo "${BLUE}⚡ Iniciando stress test (20 prescrições)...${RESET}"
	@for i in {1..20}; do \
		curl -s -X POST http://localhost:3001/api/prescricoes \
			-H "Content-Type: application/json" \
			-d "{\"id_medico\":$$((1 + RANDOM % 3)),\"id_paciente\":$$((1 + RANDOM % 5)),\"medicamentos\":[{\"id_medicamento\":$$((1 + RANDOM % 5)),\"dosagem\":\"500mg\",\"horario\":\"08:00\"}]}" \
			> /dev/null & \
	done
	@wait
	@echo "${GREEN}✅ 20 prescrições criadas${RESET}"
	@sleep 3
	@echo "${BLUE}📊 Contando registros nas views...${RESET}"
	@docker exec postgres psql -U hospital -d hospital_db \
		-c "SELECT 'View Farmácia: ' || COUNT(*) FROM view_farmacia UNION ALL SELECT 'View Prontuário: ' || COUNT(*) FROM view_prontuario_paciente;"

consumer-lag: ## Verifica lag do consumer
	@docker exec kafka kafka-consumer-groups \
		--bootstrap-server localhost:9092 \
		--group event-handler-cdc-group \
		--describe

mod: ## Atualiza dependências Go
	go mod tidy
	go mod download

build-command: ## Build da imagem Docker do Command Service
	docker build -f Dockerfile.command -t hospital-cqrs-cdc-command:latest .

build-event: ## Build da imagem Docker do Event Handler
	docker build -f Dockerfile.event -t hospital-cqrs-cdc-event:latest .

build-query: ## Build da imagem Docker do Query Service
	docker build -f Dockerfile.query -t hospital-cqrs-cdc-query:latest .

build-all: build-command build-event build-query ## Build de todas as imagens Docker

install-air: ## Instala Air (hot reload)
	@which air > /dev/null || (echo "${YELLOW}Instalando Air...${RESET}" && go install github.com/cosmtrek/air@latest)
	@echo "${GREEN}✅ Air instalado${RESET}"

.DEFAULT_GOAL := help
