"use client"

import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Clock, MessageSquare, Paperclip } from "lucide-react"

type Priority = "Alta" | "Media" | "Baja"

interface Task {
  id: string
  title: string
  description?: string
  priority: Priority
  dueDate: string
  tag?: string
  assignees?: { name: string; seed: string }[]
  comments?: number
  attachments?: number
  status: "Pendiente" | "En Progreso" | "Completado"
}

const priorityConfig: Record<Priority, { className: string; dotColor: string }> = {
  Alta: {
    className:
      "bg-red-100 text-red-700 border-red-200",
    dotColor: "bg-red-500",
  },
  Media: {
    className:
      "bg-amber-100 text-amber-700 border-amber-200",
    dotColor: "bg-amber-500",
  },
  Baja: {
    className:
      "bg-blue-100 text-blue-700 border-blue-200",
    dotColor: "bg-blue-500",
  },
}

const allTasks: Task[] = [
  // Alta priority
  {
    id: "t1",
    title: "Diseñar sistema de autenticación",
    description: "Crear wireframes y flujos de login, registro y recuperación de contraseña.",
    priority: "Alta",
    dueDate: "15 Jun 2026",
    tag: "Diseño",
    assignees: [{ name: "Ana García", seed: "ana" }, { name: "Carlos López", seed: "carlos" }],
    comments: 4,
    attachments: 2,
    status: "Pendiente",
  },
  {
    id: "t4",
    title: "Implementar dashboard analítico",
    description: "Integrar gráficos de barras y líneas con datos en tiempo real.",
    priority: "Alta",
    dueDate: "12 Jun 2026",
    tag: "Frontend",
    assignees: [{ name: "Ana García", seed: "ana" }],
    comments: 7,
    attachments: 3,
    status: "En Progreso",
  },
  {
    id: "t5",
    title: "API REST para gestión de usuarios",
    description: "Endpoints CRUD con validación y autenticación JWT.",
    priority: "Alta",
    dueDate: "14 Jun 2026",
    tag: "Backend",
    assignees: [{ name: "Carlos López", seed: "carlos" }, { name: "Luis Martínez", seed: "luis" }],
    comments: 3,
    status: "En Progreso",
  },
  {
    id: "t10",
    title: "Reunión de kickoff con el equipo",
    description: "Alineación de objetivos, cronograma y asignación inicial de responsabilidades.",
    priority: "Alta",
    dueDate: "28 May 2026",
    assignees: [{ name: "Ana García", seed: "ana" }, { name: "Carlos López", seed: "carlos" }],
    comments: 10,
    status: "Completado",
  },
  // Media priority
  {
    id: "t2",
    title: "Configurar CI/CD con GitHub Actions",
    description: "Automatizar el pipeline de build, test y deploy a producción.",
    priority: "Media",
    dueDate: "20 Jun 2026",
    tag: "DevOps",
    assignees: [{ name: "Luis Martínez", seed: "luis" }],
    comments: 2,
    status: "Pendiente",
  },
  {
    id: "t6",
    title: "Pruebas de integración E2E",
    priority: "Media",
    dueDate: "18 Jun 2026",
    assignees: [{ name: "Maria Torres", seed: "maria" }],
    comments: 1,
    status: "En Progreso",
  },
  {
    id: "t7",
    title: "Setup inicial del proyecto Next.js",
    description: "Configuración de TypeScript, Tailwind CSS, ESLint y estructura de carpetas.",
    priority: "Media",
    dueDate: "1 Jun 2026",
    tag: "Setup",
    assignees: [{ name: "Pedro Ruiz", seed: "pedro" }],
    comments: 5,
    status: "Completado",
  },
  // Baja priority
  {
    id: "t3",
    title: "Optimizar consultas de base de datos",
    priority: "Baja",
    dueDate: "30 Jun 2026",
    assignees: [{ name: "Maria Torres", seed: "maria" }, { name: "Pedro Ruiz", seed: "pedro" }],
    attachments: 1,
    status: "Pendiente",
  },
  {
    id: "t8",
    title: "Definir paleta de colores y tipografía",
    priority: "Baja",
    dueDate: "3 Jun 2026",
    tag: "Diseño",
    assignees: [{ name: "Ana García", seed: "ana" }, { name: "Maria Torres", seed: "maria" }],
    attachments: 4,
    status: "Completado",
  },
  {
    id: "t9",
    title: "Documentar arquitectura del sistema",
    priority: "Baja",
    dueDate: "5 Jun 2026",
    assignees: [{ name: "Carlos López", seed: "carlos" }],
    comments: 2,
    attachments: 1,
    status: "Completado",
  },
]

const groups: { priority: Priority; label: string }[] = [
  { priority: "Alta", label: "Prioridad Alta" },
  { priority: "Media", label: "Prioridad Media" },
  { priority: "Baja", label: "Prioridad Baja" },
]

const statusConfig: Record<Task["status"], { className: string }> = {
  Pendiente: { className: "bg-slate-100 text-slate-600 border-slate-200" },
  "En Progreso": { className: "bg-indigo-100 text-indigo-700 border-indigo-200" },
  Completado: { className: "bg-emerald-100 text-emerald-700 border-emerald-200" },
}

function MobileTaskRow({ task }: { task: Task }) {
  const s = statusConfig[task.status]

  return (
    <article className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4 shadow-sm">
      <div className="flex items-start justify-between gap-2">
        <h3 className="text-sm font-semibold leading-snug text-card-foreground flex-1">
          {task.title}
        </h3>
        <Badge
          variant="outline"
          className={cn("shrink-0 text-[11px] font-semibold px-2 py-0.5 rounded-full border", s.className)}
        >
          {task.status}
        </Badge>
      </div>

      {task.description && (
        <p className="text-xs leading-relaxed text-muted-foreground line-clamp-2">
          {task.description}
        </p>
      )}

      <div className="flex items-center justify-between pt-2 border-t border-border/60">
        <div className="flex items-center gap-1 text-muted-foreground">
          <Clock className="size-3" />
          <span className="text-[11px] font-medium">{task.dueDate}</span>
        </div>

        <div className="flex items-center gap-2">
          {(task.comments ?? 0) > 0 && (
            <div className="flex items-center gap-1 text-muted-foreground">
              <MessageSquare className="size-3" />
              <span className="text-[11px]">{task.comments}</span>
            </div>
          )}
          {(task.attachments ?? 0) > 0 && (
            <div className="flex items-center gap-1 text-muted-foreground">
              <Paperclip className="size-3" />
              <span className="text-[11px]">{task.attachments}</span>
            </div>
          )}
          {(task.assignees?.length ?? 0) > 0 && (
            <div className="flex -space-x-1.5">
              {task.assignees!.slice(0, 3).map((a) => (
                <Avatar key={a.seed} className="size-5 ring-1 ring-card">
                  <AvatarImage
                    src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${a.seed}`}
                    alt={a.name}
                  />
                  <AvatarFallback className="bg-primary/10 text-primary text-[9px] font-bold">
                    {a.name.charAt(0)}
                  </AvatarFallback>
                </Avatar>
              ))}
            </div>
          )}
        </div>
      </div>
    </article>
  )
}

export function MobileTaskList() {
  return (
    <main
      className="flex-1 overflow-y-auto bg-canvas px-4 py-5"
      aria-label="Lista de tareas por prioridad"
    >
      <div className="flex flex-col gap-6 pb-6">
        {groups.map(({ priority, label }) => {
          const tasks = allTasks.filter((t) => t.priority === priority)
          const { dotColor } = priorityConfig[priority]

          return (
            <section key={priority} aria-labelledby={`group-${priority}`}>
              <div className="mb-3 flex items-center gap-2">
                <span className={cn("size-2.5 rounded-full shrink-0", dotColor)} aria-hidden="true" />
                <h2
                  id={`group-${priority}`}
                  className="text-xs font-semibold uppercase tracking-wider text-muted-foreground"
                >
                  {label}
                </h2>
                <span className="ml-auto text-xs font-medium text-muted-foreground">
                  {tasks.length}
                </span>
              </div>

              <div className="flex flex-col gap-3">
                {tasks.map((task) => (
                  <MobileTaskRow key={task.id} task={task} />
                ))}
              </div>
            </section>
          )
        })}
      </div>
    </main>
  )
}
