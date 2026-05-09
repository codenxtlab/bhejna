# Bhejna Engineering Documentation

Welcome to the internal engineering documentation for the Bhejna Go API server. This documentation is intended for backend and infrastructure engineers to understand, maintain, and scale the system.

## Sections

### 🏗️ [Architecture](architecture/system-overview.md)
High-level system design, subsystems, and component interactions.

### 🔄 [Request Lifecycle](internal/request-lifecycle.md)
Detailed trace of message dispatch and webhook reconciliation.

### 🧵 [Concurrency Model](concurrency/goroutine-model.md)
Goroutines, worker pools, and SQLite transaction management.

### 🔌 [API Reference](api/overview.md)
Internal and external API design and payload specifications.

### 🔐 [Security & Invariants](security/authentication.md)
Auth flow, signature verification, and system invariants.

### ⚙️ [Operations](operations/deployment.md)
Deployment, environment variables, and failure recovery.

### 🚀 [Onboarding](onboarding/local-setup.md)
Guide for new engineers to get started.

---

*Last Updated: May 2026*
