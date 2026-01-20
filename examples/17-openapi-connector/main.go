// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Example 17: OpenAPI Connector - Convert REST APIs to LLM tools automatically
//
// This example demonstrates how to use the OpenAPI connector to automatically
// generate LLM-compatible tools from an OpenAPI specification.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/jllopis/kairos/pkg/connectors"
)

// Sample OpenAPI spec for a simple pet store API
const petStoreSpec = `
openapi: "3.0.0"
info:
  title: Pet Store API
  description: A simple pet store API for demonstration
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      summary: List all pets in the store
      parameters:
        - name: limit
          in: query
          description: Maximum number of pets to return
          required: false
          schema:
            type: integer
            default: 10
        - name: species
          in: query
          description: Filter by species
          required: false
          schema:
            type: string
            enum: [dog, cat, bird, fish]
      responses:
        "200":
          description: A list of pets
    post:
      operationId: createPet
      summary: Add a new pet to the store
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                  description: Pet's name
                species:
                  type: string
                  description: Pet's species
                age:
                  type: integer
                  description: Pet's age in years
              required:
                - name
                - species
      responses:
        "201":
          description: Pet created
  /pets/{id}:
    get:
      operationId: getPet
      summary: Get a specific pet by ID
      parameters:
        - name: id
          in: path
          description: Pet ID
          required: true
          schema:
            type: string
      responses:
        "200":
          description: A pet
        "404":
          description: Pet not found
`

func main() {
	// Create a mock pet store server for demonstration
	server := createMockServer()
	defer server.Close()

	fmt.Println("=== OpenAPI Connector Demo ===")
	fmt.Println()

	// Create connector from OpenAPI spec
	connector, err := connectors.NewFromBytes(
		[]byte(petStoreSpec),
		connectors.WithBaseURL(server.URL),
		connectors.WithAPIKey("demo-key", "X-API-Key"),
	)
	if err != nil {
		log.Fatalf("Failed to create connector: %v", err)
	}

	// List generated tools
	fmt.Println("Generated Tools:")
	fmt.Println("----------------")
	for _, tool := range connector.Tools() {
		fmt.Printf("• %s: %s\n", tool.Function.Name, tool.Function.Description)
		params, _ := json.MarshalIndent(tool.Function.Parameters, "  ", "  ")
		fmt.Printf("  Parameters: %s\n\n", params)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute tools
	fmt.Println("Executing Tools:")
	fmt.Println("----------------")

	// List pets
	fmt.Println("\n1. Listing all pets...")
	result, err := connector.Execute(ctx, "listPets", map[string]interface{}{
		"limit": 5,
	})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("   Result: %s\n", result)
	}

	// Create a new pet
	fmt.Println("\n2. Creating a new pet...")
	result, err = connector.Execute(ctx, "createPet", map[string]interface{}{
		"name":    "Buddy",
		"species": "dog",
		"age":     3,
	})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("   Result: %s\n", result)
	}

	// Get a specific pet
	fmt.Println("\n3. Getting pet with ID '1'...")
	result, err = connector.Execute(ctx, "getPet", map[string]interface{}{
		"id": "1",
	})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("   Result: %s\n", result)
	}

	// Using ExecuteJSON (useful when receiving tool calls from LLM)
	fmt.Println("\n4. Using ExecuteJSON with raw JSON arguments...")
	result, err = connector.ExecuteJSON(ctx, "getPet", `{"id": "2"}`)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("   Result: %s\n", result)
	}

	fmt.Println("\n✓ Demo completed!")
}

// createMockServer creates a simple in-memory pet store server
func createMockServer() *httptest.Server {
	pets := []map[string]interface{}{
		{"id": "1", "name": "Max", "species": "dog", "age": 5},
		{"id": "2", "name": "Luna", "species": "cat", "age": 3},
		{"id": "3", "name": "Tweety", "species": "bird", "age": 2},
	}
	nextID := 4

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/pets" && r.Method == "GET":
			// Filter by species if specified
			species := r.URL.Query().Get("species")
			var result []map[string]interface{}
			for _, pet := range pets {
				if species == "" || pet["species"] == species {
					result = append(result, pet)
				}
			}
			json.NewEncoder(w).Encode(result)

		case r.URL.Path == "/pets" && r.Method == "POST":
			var newPet map[string]interface{}
			json.NewDecoder(r.Body).Decode(&newPet)
			newPet["id"] = fmt.Sprintf("%d", nextID)
			nextID++
			pets = append(pets, newPet)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(newPet)

		case len(r.URL.Path) > 6 && r.URL.Path[:6] == "/pets/" && r.Method == "GET":
			id := r.URL.Path[6:]
			for _, pet := range pets {
				if pet["id"] == id {
					json.NewEncoder(w).Encode(pet)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Pet not found"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
