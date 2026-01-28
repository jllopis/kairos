# Inicio rápido

Este inicio rápido te da un primer contacto con Kairos usando el ejemplo más
simple posible. El objetivo es ejecutar un agente básico y entender el flujo
general sin entrar en detalles avanzados.

## Requisitos

- Go instalado.

## Ejecutar un agente básico

Desde la raíz del repo:

```bash
go run ./examples/01-hello-agent
```

Este ejemplo usa un LLM mock para que sea auto-contenido, así que no necesitas
modelo ni configuración adicional.

## ¿Qué estás viendo?

- Un agente se inicializa con configuración mínima.
- Se ejecuta un ciclo de entrada/salida.
- El resultado se imprime por consola.

Salida esperada:

```
Agent ID: hello-agent
Role: Greeter
Response: Hello from Kairos Agent!
```

## Siguiente paso

Si quieres un ejemplo con modelo real, prueba `examples/02-basic-agent`. Para un
flujo multiagente con A2A, planner y MCP, revisa la demo en
[Playbook de ejemplos](../examples/playbook/README.md).

## Ejemplo con modelo real (opcional)

Si tienes Ollama en local, puedes ejecutar el ejemplo básico:

```bash
go run ./examples/02-basic-agent
```

Config mínimo (ejemplo):

```json
{
  "llm": {
    "provider": "ollama",
    "ollama_base_url": "http://localhost:11434",
    "model": "qwen2.5-coder:7b-instruct-q5_K_M"
  }
}
```

Guárdalo en `./.kairos/settings.json` y ejecuta:

```bash
go run ./examples/02-basic-agent --config=./.kairos/settings.json
```
