package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"hospital-cqrs/internal/queries"
	"hospital-cqrs/pkg/database"
)

func main() {
	log.Println("Iniciando Query Service (Read Side - CQRS)...")

	// Conectar ao banco de dados
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	// Criar repositório de queries
	queryRepo := queries.NewQueryRepository(db)

	// Configurar Fiber
	app := fiber.New(fiber.Config{
		AppName: "Hospital CQRS - Query Service",
	})

	// Middlewares
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Rotas de saúde
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "query-service",
		})
	})

	// Rotas de queries
	api := app.Group("/api/v1")

	// Query Model 1: Farmácia
	farmacia := api.Group("/farmacia")

	// Listar todas as prescrições para a farmácia
	farmacia.Get("/prescricoes", func(c *fiber.Ctx) error {
		prescricoes, err := queryRepo.GetPrescricoesFarmacia(c.Context())
		if err != nil {
			log.Printf("Erro ao buscar prescrições da farmácia: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(prescricoes)
	})

	// Buscar prescrição específica para a farmácia
	farmacia.Get("/prescricoes/:id", func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "ID inválido"})
		}

		prescricao, err := queryRepo.GetPrescricaoFarmaciaByID(c.Context(), id)
		if err != nil {
			log.Printf("Erro ao buscar prescrição %d: %v", id, err)
			return c.Status(404).JSON(fiber.Map{"error": "Prescrição não encontrada"})
		}
		return c.JSON(prescricao)
	})

	// Query Model 2: Prontuário do Paciente
	prontuario := api.Group("/prontuario")

	// Buscar prontuário completo de um paciente
	prontuario.Get("/pacientes/:id", func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "ID inválido"})
		}

		prontuarioData, err := queryRepo.GetProntuarioPaciente(c.Context(), id)
		if err != nil {
			log.Printf("Erro ao buscar prontuário do paciente %d: %v", id, err)
			return c.Status(404).JSON(fiber.Map{"error": "Prontuário não encontrado"})
		}
		return c.JSON(prontuarioData)
	})

	// Iniciar servidor
	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "3001"
	}

	// Graceful shutdown
	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Erro ao iniciar servidor: %v", err)
		}
	}()

	log.Printf("Query Service rodando na porta %s", port)

	// Aguardar sinal de interrupção
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Encerrando Query Service...")
	if err := app.ShutdownWithContext(context.Background()); err != nil {
		log.Printf("Erro ao encerrar servidor: %v", err)
	}
}
