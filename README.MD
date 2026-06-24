# Taskify: Arquitectura Local-First & Realtime Sync

## 1. Resumen Ejecutivo
Taskify es un sistema de gestión operativa diseñado bajo el paradigma **Local-First**. Resuelve la problemática de latencia de red y la pérdida de datos en entornos de conectividad inestable. El sistema garantiza disponibilidad absoluta (100% offline) mediante un motor local, sincronizando el estado de forma asíncrona y bidireccional contra una infraestructura en la nube cuando la red se restablece, manteniendo coherencia eventual en múltiples clientes (PWA y Desktop).

## 2. Arquitectura del Sistema
El proyecto emplea un patrón **Sidecar** para la aplicación de escritorio y una arquitectura cliente-servidor tradicional para la aplicación web. 

- **Frontend / PWA:** React + TypeScript + Vite + Tailwind CSS (shadcn/ui).
- **Desktop Host:** Tauri (Rust) actuando como contenedor nativo.
- **Sidecar / Motor de Sincronización:** Go (Golang) incrustado en Tauri.
- **Persistencia Local:** SQLite integrado.
- **Persistencia Remota y Realtime:** Supabase (PostgreSQL) con WebSockets.

```plantuml
@startuml
skinparam componentStyle uml2

package "Desktop Client (Tauri)" {
  [React UI] as DesktopUI
  [Go Sync Motor\n(Sidecar)] as GoMotor
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