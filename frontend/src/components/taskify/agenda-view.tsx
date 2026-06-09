"use client"

import { useMemo, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { getTasks, type Task, type TaskPriority, type TaskStatus } from "@/services/taskService"
import { Clock, Circle, ChevronLeft, ChevronRight } from "lucide-react"

interface AgendaTask {
  id: string
  title: string
  priority: "Alta" | "Media" | "Baja"
  status: "Pendiente" | "En Progreso" | "Completado"
  month: number
  year: number
  day: number
  time: string
}

const priorityDot: Record<AgendaTask["priority"], string> = {
  Alta: "bg-red-500",
  Media: "bg-amber-400",
  Baja: "bg-blue-400",
}

const priorityCellBg: Record<AgendaTask["priority"], string> = {
  Alta: "bg-red-50",
  Media: "bg-amber-50",
  Baja: "bg-blue-50",
}

const priorityBadge: Record<AgendaTask["priority"], string> = {
  Alta: "bg-red-100 text-red-700 border-red-200",
  Media: "bg-amber-100 text-amber-700 border-amber-200",
  Baja: "bg-blue-100 text-blue-700 border-blue-200",
}

const statusBadge: Record<AgendaTask["status"], string> = {
  Pendiente: "bg-slate-100 text-slate-600 border-slate-200",
  "En Progreso": "bg-indigo-100 text-indigo-700 border-indigo-200",
  Completado: "bg-emerald-100 text-emerald-700 border-emerald-200",
}

const priorityLabel: Record<TaskPriority, AgendaTask["priority"]> = {
  high: "Alta",
  medium: "Media",
  low: "Baja",
}

const statusLabel: Record<TaskStatus, AgendaTask["status"]> = {
  todo: "Pendiente",
  in_progress: "En Progreso",
  done: "Completado",
}

const SHORT_DAYS = ["Lun", "Mar", "Mié", "Jue", "Vie", "Sáb", "Dom"]
const MONTH_NAMES = [
  "Enero",
  "Febrero",
  "Marzo",
  "Abril",
  "Mayo",
  "Junio",
  "Julio",
  "Agosto",
  "Septiembre",
  "Octubre",
  "Noviembre",
  "Diciembre",
]
const DAY_NAMES_FULL = [
  "Domingo",
  "Lunes",
  "Martes",
  "Miércoles",
  "Jueves",
  "Viernes",
  "Sábado",
]

const today = new Date()
const TODAY_YEAR = today.getFullYear()
const TODAY_MONTH = today.getMonth()
const TODAY_DAY = today.getDate()

function mondayFirstDow(date: Date): number {
  return (date.getDay() + 6) % 7
}

function daysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate()
}

function buildGrid(year: number, month: number): (number | null)[] {
  const firstDow = mondayFirstDow(new Date(year, month, 1))
  const total = daysInMonth(year, month)
  const cells: (number | null)[] = Array(firstDow).fill(null)

  for (let day = 1; day <= total; day++) {
    cells.push(day)
  }

  while (cells.length % 7 !== 0) {
    cells.push(null)
  }

  return cells
}

function getTasksByDay(
  tasks: AgendaTask[],
  year: number,
  month: number,
): Record<number, AgendaTask[]> {
  const map: Record<number, AgendaTask[]> = {}

  for (const task of tasks) {
    if (task.year === year && task.month === month) {
      if (!map[task.day]) {
        map[task.day] = []
      }

      map[task.day].push(task)
    }
  }

  return map
}

function parseDueDate(dueDate: string): Date | null {
  if (!dueDate.trim()) {
    return null
  }

  const dateOnlyMatch = dueDate.match(/^(\d{4})-(\d{2})-(\d{2})$/)
  const date = dateOnlyMatch
    ? new Date(
        Number(dateOnlyMatch[1]),
        Number(dateOnlyMatch[2]) - 1,
        Number(dateOnlyMatch[3]),
      )
    : new Date(dueDate)

  return Number.isNaN(date.getTime()) ? null : date
}

function formatTaskTime(dueDate: string, date: Date): string {
  if (/^\d{4}-\d{2}-\d{2}$/.test(dueDate)) {
    return "Todo el día"
  }

  return new Intl.DateTimeFormat("es", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(date)
}

function toAgendaTasks(tasks: Task[]): AgendaTask[] {
  return tasks
    .flatMap((task) => {
      const dueDate = parseDueDate(task.dueDate)

      if (!dueDate) {
        return []
      }

      return {
        id: task.id,
        title: task.title,
        priority: priorityLabel[task.priority],
        status: statusLabel[task.status],
        year: dueDate.getFullYear(),
        month: dueDate.getMonth(),
        day: dueDate.getDate(),
        time: formatTaskTime(task.dueDate, dueDate),
      }
    })
    .sort((firstTask, secondTask) => {
      const firstDate = new Date(
        firstTask.year,
        firstTask.month,
        firstTask.day,
      ).getTime()
      const secondDate = new Date(
        secondTask.year,
        secondTask.month,
        secondTask.day,
      ).getTime()

      return firstDate - secondDate
    })
}

interface MonthNavProps {
  year: number
  month: number
  taskCount: number
  onPrev: () => void
  onNext: () => void
  onToday: () => void
}

function MonthNav({
  year,
  month,
  taskCount,
  onPrev,
  onNext,
  onToday,
}: MonthNavProps) {
  const isCurrentMonth = year === TODAY_YEAR && month === TODAY_MONTH

  return (
    <div className="flex items-center justify-between border-b border-border px-5 py-3.5">
      <div className="flex items-center gap-3">
        <div>
          <h2 className="text-base font-bold leading-tight text-foreground">
            {MONTH_NAMES[month]} {year}
          </h2>
          <p className="text-xs text-muted-foreground">
            {taskCount} tarea{taskCount !== 1 ? "s" : ""} programada
            {taskCount !== 1 ? "s" : ""}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-1">
        <Button
          variant="outline"
          size="sm"
          className="h-8 px-2.5 text-xs font-semibold"
          onClick={onToday}
          disabled={isCurrentMonth}
        >
          Hoy
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="size-8"
          onClick={onPrev}
          aria-label="Mes anterior"
        >
          <ChevronLeft className="size-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="size-8"
          onClick={onNext}
          aria-label="Mes siguiente"
        >
          <ChevronRight className="size-4" />
        </Button>
      </div>
    </div>
  )
}

function CalendarGrid({
  tasks,
  year,
  month,
  onPrev,
  onNext,
  onToday,
}: {
  tasks: AgendaTask[]
  year: number
  month: number
  onPrev: () => void
  onNext: () => void
  onToday: () => void
}) {
  const grid = buildGrid(year, month)
  const tasksByDay = getTasksByDay(tasks, year, month)
  const taskCount = Object.values(tasksByDay).flat().length

  return (
    <div className="flex flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
      <MonthNav
        year={year}
        month={month}
        taskCount={taskCount}
        onPrev={onPrev}
        onNext={onNext}
        onToday={onToday}
      />

      <div className="flex items-center gap-4 border-b border-border px-5 py-2">
        {(["Alta", "Media", "Baja"] as const).map((priority) => (
          <div key={priority} className="flex items-center gap-1.5">
            <span className={cn("size-2 rounded-full", priorityDot[priority])} />
            <span className="text-xs text-muted-foreground">{priority}</span>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-7 border-b border-border">
        {SHORT_DAYS.map((day) => (
          <div
            key={day}
            className="py-2.5 text-center text-[11px] font-semibold uppercase tracking-wider text-muted-foreground"
          >
            {day}
          </div>
        ))}
      </div>

      <div className="grid grid-cols-7">
        {grid.map((day, index) => {
          const isToday =
            day === TODAY_DAY && month === TODAY_MONTH && year === TODAY_YEAR
          const dayTasks = day ? (tasksByDay[day] ?? []) : []
          const isLastRow = index >= grid.length - 7
          const isLastCol = index % 7 === 6

          return (
            <div
              key={index}
              className={cn(
                "relative flex min-h-[88px] flex-col gap-1 p-2",
                !isLastRow && "border-b border-border",
                !isLastCol && "border-r border-border",
                !day && "bg-muted/30",
                day && "cursor-default transition-colors hover:bg-muted/30",
              )}
            >
              {day && (
                <span
                  className={cn(
                    "flex size-6 items-center justify-center self-start rounded-full text-xs font-semibold",
                    isToday
                      ? "bg-primary text-primary-foreground"
                      : "text-foreground",
                  )}
                >
                  {day}
                </span>
              )}

              <div className="flex flex-col gap-0.5 overflow-hidden">
                {dayTasks.slice(0, 3).map((task) => (
                  <div
                    key={task.id}
                    title={task.title}
                    className={cn(
                      "flex items-center gap-1 rounded px-1 py-0.5",
                      priorityCellBg[task.priority],
                    )}
                  >
                    <span
                      className={cn(
                        "size-1.5 shrink-0 rounded-full",
                        priorityDot[task.priority],
                      )}
                    />
                    <span className="truncate text-[10px] font-medium leading-none text-foreground/80">
                      {task.title}
                    </span>
                  </div>
                ))}
                {dayTasks.length > 3 && (
                  <span className="pl-1 text-[10px] text-muted-foreground">
                    +{dayTasks.length - 3} más
                  </span>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function DailyAgenda({
  tasks,
  year,
  month,
  onPrev,
  onNext,
  onToday,
}: {
  tasks: AgendaTask[]
  year: number
  month: number
  onPrev: () => void
  onNext: () => void
  onToday: () => void
}) {
  const tasksByDay = getTasksByDay(tasks, year, month)
  const taskCount = Object.values(tasksByDay).flat().length
  const totalDays = daysInMonth(year, month)
  const startDay = year === TODAY_YEAR && month === TODAY_MONTH ? TODAY_DAY : 1
  const agendaDays = Array.from({ length: 7 }, (_, index) => startDay + index).filter(
    (day) => day <= totalDays,
  )

  return (
    <div className="flex flex-col gap-4 pb-6">
      <div className="flex items-center justify-between rounded-2xl border border-border bg-card px-4 py-3 shadow-sm">
        <div>
          <h2 className="text-sm font-bold text-foreground">
            {MONTH_NAMES[month]} {year}
          </h2>
          <p className="text-xs text-muted-foreground">
            {taskCount} tareas programadas
          </p>
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="outline"
            size="sm"
            className="h-7 px-2 text-xs font-semibold"
            onClick={onToday}
            disabled={year === TODAY_YEAR && month === TODAY_MONTH}
          >
            Hoy
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="size-7"
            onClick={onPrev}
            aria-label="Mes anterior"
          >
            <ChevronLeft className="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="size-7"
            onClick={onNext}
            aria-label="Mes siguiente"
          >
            <ChevronRight className="size-3.5" />
          </Button>
        </div>
      </div>

      {agendaDays.map((day) => {
        const dayTasks = tasksByDay[day] ?? []
        const isToday =
          day === TODAY_DAY && month === TODAY_MONTH && year === TODAY_YEAR
        const date = new Date(year, month, day)
        const dayName = DAY_NAMES_FULL[date.getDay()]

        return (
          <section key={day} aria-labelledby={`day-${day}`}>
            <div className="mb-2 flex items-center gap-3">
              <div
                className={cn(
                  "flex size-10 shrink-0 flex-col items-center justify-center rounded-xl",
                  isToday
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted text-muted-foreground",
                )}
              >
                <span className="text-[10px] font-semibold uppercase leading-none">
                  {dayName.slice(0, 3)}
                </span>
                <span className="text-lg font-bold leading-none">{day}</span>
              </div>
              <div>
                <p
                  className={cn(
                    "text-sm font-semibold",
                    isToday ? "text-primary" : "text-foreground",
                  )}
                >
                  {isToday ? "Hoy" : dayName}
                </p>
                <p className="text-xs text-muted-foreground">
                  {dayTasks.length === 0
                    ? "Sin tareas"
                    : `${dayTasks.length} tarea${dayTasks.length > 1 ? "s" : ""}`}
                </p>
              </div>
            </div>

            {dayTasks.length === 0 ? (
              <div className="ml-14 rounded-xl border border-dashed border-border bg-muted/20 px-4 py-3 text-xs text-muted-foreground">
                Día libre - no hay tareas programadas
              </div>
            ) : (
              <div className="ml-14 flex flex-col gap-2">
                {dayTasks.map((task) => (
                  <article
                    key={task.id}
                    className="flex items-start gap-3 rounded-xl border border-border bg-card p-3 shadow-sm"
                  >
                    <div className="flex flex-col items-center gap-1 pt-0.5">
                      <Circle
                        className={cn(
                          "size-2.5 fill-current",
                          task.priority === "Alta" && "text-red-500",
                          task.priority === "Media" && "text-amber-400",
                          task.priority === "Baja" && "text-blue-400",
                        )}
                      />
                      <div
                        className="w-px flex-1 bg-border"
                        style={{ minHeight: "16px" }}
                      />
                    </div>
                    <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                      <div className="flex items-start justify-between gap-2">
                        <h3 className="flex-1 text-sm font-semibold leading-snug text-card-foreground">
                          {task.title}
                        </h3>
                        <Badge
                          variant="outline"
                          className={cn(
                            "shrink-0 rounded-full border px-1.5 py-0.5 text-[10px] font-semibold",
                            statusBadge[task.status],
                          )}
                        >
                          {task.status}
                        </Badge>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="flex items-center gap-1 text-muted-foreground">
                          <Clock className="size-3" />
                          <span className="text-[11px] font-medium">
                            {task.time}
                          </span>
                        </div>
                        <Badge
                          variant="outline"
                          className={cn(
                            "h-4 rounded-full border px-1.5 py-0 text-[10px] font-semibold",
                            priorityBadge[task.priority],
                          )}
                        >
                          {task.priority}
                        </Badge>
                      </div>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </section>
        )
      })}
    </div>
  )
}

export function AgendaView() {
  const [year, setYear] = useState(TODAY_YEAR)
  const [month, setMonth] = useState(TODAY_MONTH)
  const {
    data: tasks = [],
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: ["tasks", "global"],
    queryFn: () => getTasks(),
    staleTime: 5 * 60 * 1000,
  })

  const agendaTasks = useMemo(() => toAgendaTasks(tasks), [tasks])
  const errorMessage =
    error instanceof Error ? error.message : "No se pudo cargar la agenda"

  function prevMonth() {
    if (month === 0) {
      setMonth(11)
      setYear((currentYear) => currentYear - 1)
      return
    }

    setMonth((currentMonth) => currentMonth - 1)
  }

  function nextMonth() {
    if (month === 11) {
      setMonth(0)
      setYear((currentYear) => currentYear + 1)
      return
    }

    setMonth((currentMonth) => currentMonth + 1)
  }

  function goToday() {
    setYear(TODAY_YEAR)
    setMonth(TODAY_MONTH)
  }

  const navProps = {
    year,
    month,
    onPrev: prevMonth,
    onNext: nextMonth,
    onToday: goToday,
  }

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center bg-canvas text-sm font-medium text-muted-foreground">
        Cargando agenda...
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex flex-1 items-center justify-center bg-canvas px-6 text-center text-sm font-medium text-red-600">
        {errorMessage}
      </div>
    )
  }

  if (agendaTasks.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center bg-canvas px-6 text-center text-sm font-medium text-muted-foreground">
        No hay tareas con fecha de vencimiento próxima
      </div>
    )
  }

  return (
    <>
      <main
        className="hidden flex-1 overflow-y-auto bg-canvas p-6 md:flex md:flex-col"
        aria-label="Vista de agenda mensual"
      >
        <CalendarGrid {...navProps} tasks={agendaTasks} />
      </main>

      <main
        className="flex flex-1 flex-col overflow-y-auto bg-canvas px-4 py-5 md:hidden"
        aria-label="Agenda de los próximos 7 días"
      >
        <DailyAgenda {...navProps} tasks={agendaTasks} />
      </main>
    </>
  )
}
