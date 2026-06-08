import { KanbanColumn } from "@/components/taskify/kanban-column"
import { Plus } from "lucide-react"
import { Button } from "@/components/ui/button"

const pendingTasks = [
  {
    id: "t1",
    title: "Diseñar sistema de autenticación",
    description: "Crear los wireframes y flujos de login, registro y recuperación de contraseña.",
    priority: "Alta" as const,
    dueDate: "15 Jun",
    tag: "Diseño",
    assignees: [{ name: "Ana García", seed: "ana" }, { name: "Carlos López", seed: "carlos" }],
    comments: 4,
    attachments: 2,
  },
  {
    id: "t2",
    title: "Configurar CI/CD con GitHub Actions",
    description: "Automatizar el pipeline de build, test y deploy a producción.",
    priority: "Media" as const,
    dueDate: "20 Jun",
    tag: "DevOps",
    assignees: [{ name: "Luis Martínez", seed: "luis" }],
    comments: 2,
  },
  {
    id: "t3",
    title: "Optimizar consultas de base de datos",
    priority: "Baja" as const,
    dueDate: "30 Jun",
    assignees: [{ name: "Maria Torres", seed: "maria" }, { name: "Pedro Ruiz", seed: "pedro" }],
    attachments: 1,
  },
]

const inProgressTasks = [
  {
    id: "t4",
    title: "Implementar dashboard analítico",
    description: "Integrar gráficos de barras y líneas con datos en tiempo real usando Recharts.",
    priority: "Alta" as const,
    dueDate: "12 Jun",
    tag: "Frontend",
    assignees: [{ name: "Ana García", seed: "ana" }],
    comments: 7,
    attachments: 3,
  },
  {
    id: "t5",
    title: "API REST para gestión de usuarios",
    description: "Endpoints CRUD con validación y autenticación JWT.",
    priority: "Alta" as const,
    dueDate: "14 Jun",
    tag: "Backend",
    assignees: [{ name: "Carlos López", seed: "carlos" }, { name: "Luis Martínez", seed: "luis" }],
    comments: 3,
  },
  {
    id: "t6",
    title: "Pruebas de integración E2E",
    priority: "Media" as const,
    dueDate: "18 Jun",
    assignees: [{ name: "Maria Torres", seed: "maria" }],
    comments: 1,
  },
]

const completedTasks = [
  {
    id: "t7",
    title: "Setup inicial del proyecto Next.js",
    description: "Configuración de TypeScript, Tailwind CSS, ESLint y estructura de carpetas.",
    priority: "Media" as const,
    dueDate: "1 Jun",
    tag: "Setup",
    assignees: [{ name: "Pedro Ruiz", seed: "pedro" }],
    comments: 5,
  },
  {
    id: "t8",
    title: "Definir paleta de colores y tipografía",
    priority: "Baja" as const,
    dueDate: "3 Jun",
    tag: "Diseño",
    assignees: [{ name: "Ana García", seed: "ana" }, { name: "Maria Torres", seed: "maria" }],
    attachments: 4,
  },
  {
    id: "t9",
    title: "Documentar arquitectura del sistema",
    priority: "Baja" as const,
    dueDate: "5 Jun",
    assignees: [{ name: "Carlos López", seed: "carlos" }],
    comments: 2,
    attachments: 1,
  },
  {
    id: "t10",
    title: "Reunión de kickoff con el equipo",
    description: "Alineación de objetivos, cronograma y asignación inicial de responsabilidades.",
    priority: "Alta" as const,
    dueDate: "28 May",
    assignees: [
      { name: "Ana García", seed: "ana" },
      { name: "Carlos López", seed: "carlos" },
      { name: "Luis Martínez", seed: "luis" },
    ],
    comments: 10,
  },
]

const columns = [
  {
    id: "pending",
    title: "Pendiente",
    tasks: pendingTasks,
    accentColor: "bg-slate-200 text-slate-600 dark:bg-slate-700 dark:text-slate-300",
    dotColor: "bg-slate-400",
  },
  {
    id: "in-progress",
    title: "En Progreso",
    tasks: inProgressTasks,
    accentColor: "bg-indigo-100 text-indigo-700 dark:bg-indigo-900/50 dark:text-indigo-300",
    dotColor: "bg-indigo-500",
  },
  {
    id: "completed",
    title: "Completado",
    tasks: completedTasks,
    accentColor: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300",
    dotColor: "bg-emerald-500",
  },
]

export function KanbanBoard() {
  return (
    <main
      className="flex-1 overflow-x-auto kanban-scroll bg-canvas"
      aria-label="Tablero Kanban"
    >
      <div className="flex h-full gap-4 p-5 md:p-6">
        {columns.map((col) => (
          <KanbanColumn
            key={col.id}
            title={col.title}
            tasks={col.tasks}
            accentColor={col.accentColor}
            dotColor={col.dotColor}
          />
        ))}

        {/* Add Column Button */}
        <div className="flex shrink-0 items-start pt-0.5">
          <Button
            variant="outline"
            className="h-12 w-44 gap-2 border-dashed border-border/80 text-muted-foreground hover:text-foreground hover:border-border hover:bg-column"
          >
            <Plus className="size-4" />
            Nueva columna
          </Button>
        </div>
      </div>
    </main>
  )
}
