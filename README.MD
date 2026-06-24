# Taskify - Local-First Architecture & Realtime Synchronization

## 📖 Executive Summary

Taskify is an operational management system built under the **Local-First** paradigm.

The platform addresses network latency, intermittent connectivity, and data-loss challenges by prioritizing local persistence and offline operation.

Key characteristics:

- ✅ 100% offline availability.
- ✅ Local-first data ownership.
- ✅ Bidirectional asynchronous synchronization.
- ✅ Eventual consistency across multiple clients.
- ✅ Realtime updates through WebSockets.
- ✅ Cross-platform support (Desktop + PWA).

The system maintains a local source of truth and synchronizes state with cloud infrastructure whenever connectivity becomes available.

---

# 🏗️ System Architecture

Taskify uses a hybrid architecture:

- **Desktop Application:** Sidecar Pattern
- **Web Application (PWA):** Traditional Client-Server Architecture

## Technology Stack

| Layer | Technology |
|---------|------------|
| Frontend | React + TypeScript + Vite |
| UI Components | Tailwind CSS + shadcn/ui |
| Desktop Host | Tauri (Rust) |
| Synchronization Engine | Go (Golang) |
| Local Database | SQLite |
| Cloud Database | PostgreSQL (Supabase) |
| Realtime Layer | Supabase Realtime (WebSockets) |

---

## Architecture Diagram

```plantuml
@startuml
skinparam componentStyle uml2

package "Desktop Client (Tauri)" {
  [React UI] as DesktopUI
  [Go Sync Engine\n(Sidecar)] as GoMotor
  database "SQLite\n(Local State)" as SQLite

  DesktopUI <--> GoMotor : gRPC / HTTP
  GoMotor <--> SQLite : Read/Write
}

package "Web Client (PWA)" {
  [React UI] as WebUI
}

cloud "Cloud Infrastructure" {
  database "PostgreSQL\n(Supabase)" as Postgres
  [Realtime Server\n(WebSockets)] as WS
}

GoMotor <--> Postgres : Async Sync (Pull/Push)
WebUI <--> Postgres : REST API
WebUI <--> WS : Subscribe (Events)
WS --> GoMotor : Broadcast

@enduml
```

---

# 🔄 Synchronization Flow

```text
┌─────────────┐
│ React UI    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Go Sidecar  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ SQLite      │
└──────┬──────┘
       │
       │ Sync when online
       ▼
┌─────────────┐
│ PostgreSQL  │
│ Supabase    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Realtime WS │
└─────────────┘
```

---

# 📐 Architectural Decision Records (ADRs)

## ADR-001: Conflict Resolution Strategy (Last Write Wins)

### Context

In a Local-First environment, multiple clients may modify the same entity while disconnected from the network.

### Decision

Implement a **Last Write Wins (LWW)** conflict resolution strategy using UTC timestamps (`updated_at`).

### Benefits

- Simple implementation.
- Predictable synchronization behavior.
- High availability.
- Better user experience during offline operation.

### Trade-off

The system favors:

- Availability
- Responsiveness
- Offline operation

over:

- Immediate strong consistency
- Distributed transaction coordination

---

## ADR-002: Soft Deletes for Synchronization Integrity

### Context

Physical deletions (`DELETE`) generated synchronization race conditions because removed records left no synchronization trail.

### Decision

Replace hard deletes with logical deletions using:

```sql
deleted_at TIMESTAMP NULL
```

### Benefits

- Deterministic synchronization.
- Full auditability.
- Historical recovery.
- Simplified conflict resolution.

### Consequences

All domain queries must filter logically deleted records.

---

## ADR-003: Production Credential Isolation

### Context

Desktop binaries distribute executable code directly to end users.

Exposing infrastructure credentials inside the application package creates significant security risks.

### Decision

Avoid shipping static environment secrets.

Configuration is injected securely during the build and deployment process.

### Benefits

- Reduced attack surface.
- Safer enterprise deployments.
- Better secret management practices.

---

# 🔐 Security

## Row Level Security (RLS)

The cloud database uses PostgreSQL Row Level Security policies to guarantee tenant isolation.

Features:

- JWT-based authentication.
- Tenant-aware authorization.
- Secure realtime subscriptions.
- Data isolation between organizations.

---

## Realtime Security

WebSocket payloads are protected through authenticated Supabase channels.

Security goals:

- Prevent unauthorized subscriptions.
- Prevent tenant data leakage.
- Maintain end-to-end authorization enforcement.

---

# 🎨 User Experience (UX)

The user interface follows established usability principles inspired by Nielsen's heuristics.

### Design Goals

- Immediate feedback.
- Offline visibility.
- Synchronization transparency.
- Responsive interactions.
- Minimal user disruption.

### Examples

- Network status indicators.
- Synchronization progress feedback.
- Automatic reconnection handling.
- Light/Dark theme transitions.
- Optimistic UI updates.

---

# ⚡ Key Quality Attributes

| Attribute | Strategy |
|------------|-----------|
| Availability | Local SQLite persistence |
| Reliability | Asynchronous synchronization |
| Performance | Local reads and writes |
| Scalability | Event-driven synchronization |
| Security | RLS + JWT |
| Maintainability | Sidecar architecture |
| Portability | Tauri Desktop + PWA |
| Usability | Offline-first UX |

---

# 🚀 Build & Deployment Guide

## Prerequisites

### Node.js

```bash
>= 18.x
```

### Go

```bash
>= 1.21
```

### Rust

Required for:

- Cargo
- Tauri

Install from:

https://www.rust-lang.org/tools/install

---

# 🖥️ Production Build (Windows x64)

The current build process uses a manual pipeline (pre-CI/CD).

Goals:

- Optimize executable size.
- Hide sidecar terminal window.
- Improve startup performance.

---

## Step 1: Build the Go Synchronization Engine

```bash
cd backend

go build \
  -ldflags="-H windowsgui -s -w" \
  -o ../frontend/src-tauri/bin/backend-x86_64-pc-windows-msvc.exe \
  ./cmd
```

### Flags Explained

| Flag | Purpose |
|--------|----------|
| `-H windowsgui` | Hides console window |
| `-s` | Removes symbol table |
| `-w` | Removes debug information |

---

## Step 2: Package Desktop Application

```bash
cd ../frontend

npm install

npm run tauri build
```

Generated artifacts:

```text
frontend/src-tauri/target/release/bundle/
```

---

# 📦 Deployment Targets

- Windows Desktop (.msi)
- Windows Desktop (.exe)
- Progressive Web App (PWA)

Future targets:

- macOS
- Linux
- Self-hosted synchronization backend

---

# 🌟 Core Features

- Local-First Architecture
- Offline Operation
- Realtime Synchronization
- SQLite Local Database
- PostgreSQL Cloud Database
- WebSocket Updates
- Desktop Application (Tauri)
- Progressive Web App (PWA)
- Multi-Client Synchronization
- Eventual Consistency
- Conflict Resolution (LWW)
- Soft Delete Synchronization
- JWT Authentication
- Row Level Security (RLS)

---

# 📄 License

This project is distributed under the terms defined in the repository license.