---
description: Guide for architecting complex software features using clean architecture principles
---
1. Requirement Analysis & Domain Modeling
   - **Define Capabilities**: List succinct feature requirements.
   - **Identify Entities**: Define core data structures in `common/` or `types/`.
   - **Define Interfaces**: Create Go interfaces for external dependencies (DB, API, Net) to allow mocking.

2. Modular Architecture Design
   - **Layer Separation**:
     - `cmd/`: CLI entrypoints (Controller layer).
     - `core/` or `engine/`: Business logic (Service layer).
     - `utils/` or `pkg/`: Reusable helpers.
   - **Dependency Injection**: Pass dependencies (Config, Logger, DB) via struct fields or constructors, avoid global states where possible.

3. Concurrency Strategy
   - **Worker Pool Pattern**: Use for high-throughput tasks (scanning, spidering).
     - Define `Worker` struct with Input/Output channels.
     - Use `sync.WaitGroup` for lifecycle management.
   - **Context Propagation**: Always pass `context.Context` for cancellation and timeouts.

4. Error Handling & Resilience
   - **Typed Errors**: Define custom error types for domain logic.
   - **Retry Mechanisms**: Implement exponential backoff for network ops.
   - **Resource cleanup**: Use `defer` for closing handles (Files, Rows, Connections).

5. Testability Checklist
   - **Unit Tests**: Test core logic with mocked interfaces.
   - **Table-Driven Tests**: Use for validation logic and parser tests.
   - **Integration Tests**: Verify interactions between components (e.g. Spider -> DB).

6. Code Review Standard (Self-Correction)
   - Is the function too long? -> Split it.
   - Is the struct too fat? -> Compose it.
   - Are magic numbers used? -> constantize them.
