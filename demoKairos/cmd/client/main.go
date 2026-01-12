package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jllopis/kairos/demoKairos/internal/demo"
	"github.com/jllopis/kairos/pkg/a2a/client"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	var (
		addr           = flag.String("addr", "localhost:9030", "orchestrator gRPC endpoint")
		query          = flag.String("q", "", "user query")
		timeoutSeconds = flag.Int("timeout", 60, "client timeout in seconds")
	)
	flag.Parse()

	text := strings.TrimSpace(*query)
	if text == "" {
		fmt.Print("Consulta: ")
		input, _ := os.ReadFile("/dev/stdin")
		text = strings.TrimSpace(string(input))
	}
	if text == "" {
		log.Fatal("query is required")
	}

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	opts := []client.Option{}
	if *timeoutSeconds > 0 {
		opts = append(opts, client.WithTimeout(time.Duration(*timeoutSeconds)*time.Second))
	}
	cli := client.New(conn, opts...)
	ctx := context.Background()
	contextID := uuid.NewString()
	msg := demo.NewTextMessage(a2av1.Role_ROLE_USER, text, contextID, "")
	stream, err := cli.SendStreamingMessage(ctx, &a2av1.SendMessageRequest{Request: msg})
	if err != nil {
		log.Fatalf("send: %v", err)
	}

	for {
		event, err := stream.Recv()
		if err != nil {
			log.Printf("stream ended: %v", err)
			break
		}
		switch payload := event.Payload.(type) {
		case *a2av1.StreamResponse_Task:
			fmt.Printf("[task] id=%s context=%s\n", payload.Task.GetId(), payload.Task.GetContextId())
		case *a2av1.StreamResponse_StatusUpdate:
			status := payload.StatusUpdate
			fmt.Printf("[status] %s - %s\n", eventType(status.GetMetadata()), extractText(status.GetStatus().GetMessage()))
		case *a2av1.StreamResponse_Msg:
			text := extractText(payload.Msg)
			if text != "" {
				fmt.Printf("[msg] %s\n", text)
			}
		case *a2av1.StreamResponse_ArtifactUpdate:
			fmt.Printf("[artifact] %s\n", payload.ArtifactUpdate.GetArtifact().GetName())
		}
	}
}

func extractText(msg *a2av1.Message) string {
	if msg == nil {
		return ""
	}
	for _, part := range msg.Parts {
		if part == nil {
			continue
		}
		if text := part.GetText(); text != "" {
			return text
		}
	}
	return ""
}

func eventType(meta *structpb.Struct) string {
	if meta == nil {
		return "status"
	}
	value, ok := meta.AsMap()["event_type"].(string)
	if ok && value != "" {
		return value
	}
	return "status"
}
