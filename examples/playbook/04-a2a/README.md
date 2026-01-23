# Playbook 14 - Discovery & A2A Communication

Goal: make agents discoverable and enable peer-to-peer communication.

## Why this step?

In a real-world ecosystem, agents don't work in isolation. A "Flight Agent" needs to be found by a "Concierge Agent". **A2A (Agent-to-Agent)** communication provides a standardized way for agents to describe their capabilities (Agent Cards) and talk to each other over the network.

## SkyGuide Narrative

SkyGuide is no longer a lonely script. We are making it a "Service". It will now have an **Agent Card** and will start an **A2A Server** so other specialist agents can find it and collaborate with it.

## Incremental reuse

- Add `internal/a2a` for server/client wiring and discovery helpers.

## What to implement

- Define an **AgentCard** for SkyGuide (`pkg/a2a/agentcard`).
- Start an **A2A Server** (`pkg/a2a/server`) supporting `httpjson` or `jsonrpc`.
- Use `pkg/discovery` to register the agent (mock or local discovery).
- Create an **A2A Client** (`pkg/a2a/client`) to talk to your own server as a test.
- Reuse provider/config wiring from step 02 via shared helpers.

## Suggested checks

- The A2A server starts and listens on a port.
- The Agent Card is served and contains the correct role/capabilities.
- A client can send a message and receive the agent's response over the network.

## Manual tests

- Use `curl` or a custom client to send a "Hello" to the agent's A2A endpoint.

## Expected behavior

- The server receives the request, processes it via the internal agent, and returns the response in the selected A2A protocol format.

## Checklist

- [ ] Agent Card is correctly populated.
- [ ] A2A Server handles requests concurrently.
- [ ] Discovery registration is successful.

## References

- `pkg/a2a`
- `pkg/discovery`
- `pkg/a2a/agentcard`
