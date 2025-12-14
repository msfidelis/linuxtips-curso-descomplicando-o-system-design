package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"hospital-cqrs/internal/commands"
	"hospital-cqrs/internal/domain"
	"hospital-cqrs/pkg/database"
	"hospital-cqrs/pkg/kafka"
)

func main() {
	log.Println("Iniciando Command Service (Write Side - CQRS)...")

	// Conectar ao banco de dados
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	// Criar produtor Kafka
	producer, err := kafka.NewProducer("prescricoes")
	if err != nil {
		log.Fatalf("Erro ao criar produtor Kafka: %v", err)
	}
	defer producer.Close()

	// Criar handler de comandos
	prescricaoHandler := commands.NewPrescricaoHandler(db, producer)

	// Configurar Fiber
	app := fiber.New(fiber.Config{
		AppName: "Hospital CQRS - Command Service",
	})

	// Middlewares
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Rotas de saúde
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "command-service",
		})
	})

	// Rotas de comandos
	api := app.Group("/api/v1")

	// Endpoints auxiliares (para consulta rápida - não são queries do CQRS)
	api.Get("/medicos", func(c *fiber.Ctx) error {
		medicos, err := prescricaoHandler.ListMedicos(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(medicos)
	})

	api.Get("/pacientes", func(c *fiber.Ctx) error {
		pacientes, err := prescricaoHandler.ListPacientes(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(pacientes)
	})

	api.Get("/medicamentos", func(c *fiber.Ctx) error {
		medicamentos, err := prescricaoHandler.ListMedicamentos(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(medicamentos)
	})

	// Comando: Criar Prescrição (Write Side)
	api.Post("/prescricoes", func(c *fiber.Ctx) error {
		var dto domain.CriarPrescricaoDTO
		if err := c.BodyParser(&dto); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Dados inválidos"})
		}

		prescricao, err := prescricaoHandler.CriarPrescricao(c.Context(), dto)
		if err != nil {
			log.Printf("Erro ao criar prescrição: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(201).JSON(fiber.Map{
			"message":    "Prescrição criada com sucesso",
			"prescricao": prescricao,
		})
	})

	// Iniciar servidor
	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "3000"
	}

	// Graceful shutdown
	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Erro ao iniciar servidor: %v", err)
		}
	}()

	log.Printf("Command Service rodando na porta %s", port)

	// Aguardar sinal de interrupção
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Encerrando Command Service...")
	if err := app.ShutdownWithContext(context.Background()); err != nil {
		log.Printf("Erro ao encerrar servidor: %v", err)
	}
}
