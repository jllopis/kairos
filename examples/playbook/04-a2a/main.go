package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jllopis/kairos/examples/playbook/shared/a2a"
	"github.com/jllopis/kairos/examples/playbook/shared/agent"
	"github.com/jllopis/kairos/examples/playbook/shared/config"
	"github.com/jllopis/kairos/examples/playbook/shared/observability"
	a2aclient "github.com/jllopis/kairos/pkg/a2a/httpjson/client"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/providers/gemini"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.MustLoad()
	shutdown, err := observability.Init("skyguide-a2a", "1.0.0", cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown(context.Background())

	provider, err := gemini.NewWithAPIKey(ctx, cfg.LLM.APIKey)
	if err != nil {
		log.Fatal(err)
	}

	ag, err := agent.NewSkyGuide("skyguide", provider, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer ag.Close()

	// Tarjeta de Identidad del Agente
	card := &a2av1.AgentCard{
		Name:        "SkyGuide Concierge",
		Description: "Asistente de viajes SkyGuide desponible como servicio A2A.",
		Version:     "1.0.0",
	}

	// Iniciar servidor A2A
	addr := ":8080"
	if err := a2a.RunServer(ctx, addr, ag, card); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n--- Simulando llamada A2A de otro sistema ---")
	time.Sleep(1 * time.Second) // esperar a que el servidor esté listo

	// Cliente A2A (simulando otro agente o sistema externo)
	client := a2aclient.New("http://localhost:8080")
	resp, err := client.SendMessage(ctx, &a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			MessageId: uuid.New().String(),
			Role:      a2av1.Role_ROLE_USER,
			Parts: []*a2av1.Part{
				{
					Part: &a2av1.Part_Text{Text: "¿Qué servicios ofreces como SkyGuide?"},
				},
			},
		},
		Configuration: &a2av1.SendMessageConfiguration{
			Blocking: true, // Forzamos a esperar la respuesta síncrona
		},
	})
	if err != nil {
		log.Printf("Error A2A: %v", err)
	} else if msg := resp.GetMsg(); msg != nil {
		fmt.Println("SkyGuide responde via A2A:")
		for _, part := range msg.GetParts() {
			if text := part.GetText(); text != "" {
				fmt.Println(text)
			}
		}
	} else if task := resp.GetTask(); task != nil {
		fmt.Printf("\nLa tarea se ha iniciado de forma asíncrona (ID: %s). Para ver el resultado deberíamos consultar el estado de la tarea.\n", task.GetId())
	}

	<-ctx.Done()
	fmt.Println("\nDeteniendo SkyGuide...")
}
