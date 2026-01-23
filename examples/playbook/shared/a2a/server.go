package a2a

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jllopis/kairos/pkg/a2a/httpjson"
	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/core"
)

// RunServer inicia un servidor A2A HTTP+JSON para un agente
func RunServer(ctx context.Context, addr string, agent core.Agent, card *a2av1.AgentCard) error {
	// 1. Crear el handler de a2a para el agente
	handler := server.NewAgentHandler(agent, server.WithAgentCard(card))

	// 2. Envolverlo en un servidor HTTP+JSON
	httpServer := httpjson.New(handler)

	// 3. Configurar el servidor HTTP estándar
	mux := http.NewServeMux()
	mux.Handle("/", httpServer)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	fmt.Printf("A2A Server listening on %s\n", addr)

	// Iniciar el servidor en una goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("A2A Server error: %v\n", err)
		}
	}()

	return nil
}
