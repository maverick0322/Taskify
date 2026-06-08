"use client"

import { useState } from "react"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Clock, Circle, ChevronLeft, ChevronRight } from "lucide-react"

// ─── Static task data pinned to June 2026 ────────────────────────────────────

interface AgendaTask {
  id: string
  title: string
  priority: "Alta" | "Media" | "Baja"
  status: "Pendiente" | "En Progreso" | "Completado"
  month: number // 0-indexed
  year: number
  day: number
  time: string
}

const tasks: AgendaTask[] = [
  { id: "a1",  title: "Reunión de kickoff",        priority: "Alta",  status: "En Progreso", year: 2026, month: 5, day: 2,  time: "09:00" },
  { id: "a2",  title: "Diseñar pantallas de login", priority: "Alta",  status: "Pendiente",   year: 2026, month: 5, day: 5,  time: "10:30" },
  { id: "a3",  title: "Configurar CI/CD",           priority: "Media", status: "Pendiente",   year: 2026, month: 5, day: 5,  time: "14:00" },
  { id: "a4",  title: "Revisión de diseño UI",      priority: "Media", status: "Completado",  year: 2026, month: 5, day: 9,  time: "11:00" },
  { id: "a5",  title: "Sprint planning Q3",         priority: "Alta",  status: "En Progreso", year: 2026, month: 5, day: 10, time: "09:00" },
  { id: "a6",  title: "API de usuarios",            priority: "Alta",  status: "En Progreso", year: 2026, month: 5, day: 12, time: "16:00" },
  { id: "a7",  title: "Dashboard analítico",        priority: "Alta",  status: "En Progreso", year: 2026, month: 5, day: 14, time: "10:00" },
  { id: "a8",  title: "Pruebas E2E",                priority: "Media", status: "Pendiente",   year: 2026, month: 5, day: 14, time: "15:00" },
  { id: "a9",  title: "Autenticación JWT",          priority: "Alta",  status: "Pendiente",   year: 2026, month: 5, day: 15, time: "09:30" },
  { id: "a10", title: "Code review backend",        priority: "Media", status: "Pendiente",   year: 2026, month: 5, day: 18, time: "11:00" },
  { id: "a11", title: "Optimizar consultas DB",     priority: "Baja",  status: "Pendiente",   year: 2026, month: 5, day: 20, time: "14:00" },
  { id: "a12", title: "Documentar arquitectura",   priority: "Baja",  status: "Pendiente",   year: 2026, month: 5, day: 23, time: "10:00" },
  { id: "a13", title: "Demo al cliente",            priority: "Alta",  status: "Pendiente",   year: 2026, month: 5, day: 25, time: "17:00" },
  { id: "a14", title: "Retrospectiva del sprint",   priority: "Media", status: "Pendiente",   year: 2026, month: 5, day: 27, time: "09:00" },
  { id: "a15", title: "Despliegue a producción",   priority: "Alta",  status: "Pendiente",   year: 2026, month: 5, day: 30, time: "18:00" },
]

// ─── Style maps ───────────────────────────────────────────────────────────────

const priorityDot: Record<AgendaTask["priority"], string> = {
  Alta:  "bg-red-500",
  Media: "bg-amber-400",
  Baja:  "bg-blue-400",
}

const priorityCellBg: Record<AgendaTask["priority"], string> = {
  Alta:  "bg-red-50",
  Media: "bg-amber-50",
  Baja:  "bg-blue-50",
}

const priorityBadge: Record<AgendaTask["priority"], string> = {
  Alta:  "bg-red-100 text-red-700 border-red-200",
  Media: "bg-amber-100 text-amber-700 border-amber-200",
  Baja:  "bg-blue-100 text-blue-700 border-blue-200",
}

const statusBadge: Record<AgendaTask["status"], string> = {
  "Pendiente":   "bg-slate-100 text-slate-600 border-slate-200",
  "En Progreso": "bg-indigo-100 text-indigo-700 border-indigo-200",
  "Completado":  "bg-emerald-100 text-emerald-700 border-emerald-200",
}

// ─── Calendar helpers ─────────────────────────────────────────────────────────

const SHORT_DAYS   = ["Lun", "Mar", "Mié", "Jue", "Vie", "Sáb", "Dom"]
const MONTH_NAMES  = [
  "Enero","Febrero","Marzo","Abril","Mayo","Junio",
  "Julio","Agosto","Septiembre","Octubre","Noviembre","Diciembre",
]
const DAY_NAMES_FULL = ["Domingo","Lunes","Martes","Miércoles","Jueves","Viernes","Sábado"]

// Monday-first day-of-week index (0=Mon … 6=Sun)
function mondayFirstDow(date: Date): number {
  return (date.getDay() + 6) % 7
}

function daysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate()
}

function buildGrid(year: number, month: number): (number | null)[] {
  const firstDow = mondayFirstDow(new Date(year, month, 1))
  const total    = daysInMonth(year, month)
  const cells: (number | null)[] = Array(firstDow).fill(null)
  for (let d = 1; d <= total; d++) cells.push(d)
  while (cells.length % 7 !== 0) cells.push(null)
  return cells
}

function getTasksByDay(year: number, month: number): Record<number, AgendaTask[]> {
  const map: Record<number, AgendaTask[]> = {}
  for (const t of tasks) {
    if (t.year === year && t.month === month) {
      if (!map[t.day]) map[t.day] = []
      map[t.day].push(t)
    }
  }
  return map
}

// Today is pinned to June 7, 2026 (project's static current date)
const TODAY_YEAR  = 2026
const TODAY_MONTH = 5
const TODAY_DAY   = 7

// ─── Month Navigation Header (shared) ────────────────────────────────────────

interface MonthNavProps {
  year: number
  month: number
  taskCount: number
  onPrev: () => void
  onNext: () => void
  onToday: () => void
}

function MonthNav({ year, month, taskCount, onPrev, onNext, onToday }: MonthNavProps) {
  const isCurrentMonth = year === TODAY_YEAR && month === TODAY_MONTH

  return (
    <div className="flex items-center justify-between border-b border-border px-5 py-3.5">
      {/* Left: title + count */}
      <div className="flex items-center gap-3">
        <div>
          <h2 className="text-base font-bold text-foreground leading-tight">
            {MONTH_NAMES[month]} {year}
          </h2>
          <p className="text-xs text-muted-foreground">
            {taskCount} tarea{taskCount !== 1 ? "s" : ""} programada{taskCount !== 1 ? "s" : ""}
          </p>
        </div>
      </div>

      {/* Right: navigation */}
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

// ─── Desktop: Monthly Calendar Grid ─────────────────────────────────────────

function CalendarGrid({
  year, month, onPrev, onNext, onToday,
}: {
  year: number; month: number
  onPrev: () => void; onNext: () => void; onToday: () => void
}) {
  const grid       = buildGrid(year, month)
  const tasksByDay = getTasksByDay(year, month)
  const taskCount  = Object.values(tasksByDay).flat().length

  return (
    <div className="flex flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
      {/* Month nav header */}
      <MonthNav
        year={year} month={month} taskCount={taskCount}
        onPrev={onPrev} onNext={onNext} onToday={onToday}
      />

      {/* Priority legend */}
      <div className="flex items-center gap-4 border-b border-border px-5 py-2">
        {(["Alta", "Media", "Baja"] as const).map((p) => (
          <div key={p} className="flex items-center gap-1.5">
            <span className={cn("size-2 rounded-full", priorityDot[p])} />
            <span className="text-xs text-muted-foreground">{p}</span>
          </div>
        ))}
      </div>

      {/* Day-of-week headers */}
      <div className="grid grid-cols-7 border-b border-border">
        {SHORT_DAYS.map((d) => (
          <div
            key={d}
            className="py-2.5 text-center text-[11px] font-semibold uppercase tracking-wider text-muted-foreground"
          >
            {d}
          </div>
        ))}
      </div>

      {/* Grid cells */}
      <div className="grid grid-cols-7">
        {grid.map((day, idx) => {
          const isToday    = day === TODAY_DAY && month === TODAY_MONTH && year === TODAY_YEAR
          const dayTasks   = day ? (tasksByDay[day] ?? []) : []
          const isLastRow  = idx >= grid.length - 7
          const isLastCol  = idx % 7 === 6

          return (
            <div
              key={idx}
              className={cn(
                "relative flex min-h-[88px] flex-col gap-1 p-2",
                !isLastRow && "border-b border-border",
                !isLastCol && "border-r border-border",
                !day && "bg-muted/30",
                day && "cursor-default transition-colors hover:bg-muted/30"
              )}
            >
              {day && (
                <span
                  className={cn(
                    "flex size-6 items-center justify-center self-start rounded-full text-xs font-semibold",
                    isToday
                      ? "bg-primary text-primary-foreground"
                      : "text-foreground"
                  )}
                >
                  {day}
                </span>
              )}

              <div className="flex flex-col gap-0.5 overflow-hidden">
                {dayTasks.slice(0, 3).map((t) => (
                  <div
                    key={t.id}
                    title={t.title}
                    className={cn(
                      "flex items-center gap-1 rounded px-1 py-0.5",
                      priorityCellBg[t.priority]
                    )}
                  >
                    <span className={cn("size-1.5 shrink-0 rounded-full", priorityDot[t.priority])} />
                    <span className="truncate text-[10px] font-medium leading-none text-foreground/80">
                      {t.title}
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

// ─── Mobile: Daily Agenda (next 7 days from today or month start) ─────────────

function DailyAgenda({
  year, month, onPrev, onNext, onToday,
}: {
  year: number; month: number
  onPrev: () => void; onNext: () => void; onToday: () => void
}) {
  const tasksByDay  = getTasksByDay(year, month)
  const taskCount   = Object.values(tasksByDay).flat().length
  const totalDays   = daysInMonth(year, month)

  // For the current month start from today; otherwise show first 7 days
  const startDay    = (year === TODAY_YEAR && month === TODAY_MONTH) ? TODAY_DAY : 1
  const agendaDays  = Array.from({ length: 7 }, (_, i) => startDay + i).filter((d) => d <= totalDays)

  return (
    <div className="flex flex-col gap-4 pb-6">
      {/* Mobile month nav header */}
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
          <Button variant="ghost" size="icon" className="size-7" onClick={onPrev} aria-label="Mes anterior">
            <ChevronLeft className="size-3.5" />
          </Button>
          <Button variant="ghost" size="icon" className="size-7" onClick={onNext} aria-label="Mes siguiente">
            <ChevronRight className="size-3.5" />
          </Button>
        </div>
      </div>

      {/* Days list */}
      {agendaDays.map((day) => {
        const dayTasks = tasksByDay[day] ?? []
        const isToday  = day === TODAY_DAY && month === TODAY_MONTH && year === TODAY_YEAR
        const date     = new Date(year, month, day)
        const dayName  = DAY_NAMES_FULL[date.getDay()]

        return (
          <section key={day} aria-labelledby={`day-${day}`}>
            <div className="mb-2 flex items-center gap-3">
              <div
                className={cn(
                  "flex size-10 shrink-0 flex-col items-center justify-center rounded-xl",
                  isToday ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
                )}
              >
                <span className="text-[10px] font-semibold uppercase leading-none">
                  {dayName.slice(0, 3)}
                </span>
                <span className="text-lg font-bold leading-none">{day}</span>
              </div>
              <div>
                <p className={cn("text-sm font-semibold", isToday ? "text-primary" : "text-foreground")}>
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
                Día libre — no hay tareas programadas
              </div>
            ) : (
              <div className="ml-14 flex flex-col gap-2">
                {dayTasks.map((t) => (
                  <article
                    key={t.id}
                    className="flex items-start gap-3 rounded-xl border border-border bg-card p-3 shadow-sm"
                  >
                    <div className="flex flex-col items-center gap-1 pt-0.5">
                      <Circle
                        className={cn(
                          "size-2.5 fill-current",
                          t.priority === "Alta"  && "text-red-500",
                          t.priority === "Media" && "text-amber-400",
                          t.priority === "Baja"  && "text-blue-400",
                        )}
                      />
                      <div className="w-px flex-1 bg-border" style={{ minHeight: "16px" }} />
                    </div>
                    <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                      <div className="flex items-start justify-between gap-2">
                        <h3 className="flex-1 text-sm font-semibold leading-snug text-card-foreground">
                          {t.title}
                        </h3>
                        <Badge
                          variant="outline"
                          className={cn(
                            "shrink-0 rounded-full border px-1.5 py-0.5 text-[10px] font-semibold",
                            statusBadge[t.status]
                          )}
                        >
                          {t.status}
                        </Badge>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="flex items-center gap-1 text-muted-foreground">
                          <Clock className="size-3" />
                          <span className="text-[11px] font-medium">{t.time}</span>
                        </div>
                        <Badge
                          variant="outline"
                          className={cn(
                            "h-4 rounded-full border px-1.5 py-0 text-[10px] font-semibold",
                            priorityBadge[t.priority]
                          )}
                        >
                          {t.priority}
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

// ─── Export ───────────────────────────────────────────────────────────────────

export function AgendaView() {
  const [year,  setYear]  = useState(TODAY_YEAR)
  const [month, setMonth] = useState(TODAY_MONTH)

  function prevMonth() {
    if (month === 0) { setMonth(11); setYear((y) => y - 1) }
    else             { setMonth((m) => m - 1) }
  }

  function nextMonth() {
    if (month === 11) { setMonth(0); setYear((y) => y + 1) }
    else              { setMonth((m) => m + 1) }
  }

  function goToday() {
    setYear(TODAY_YEAR)
    setMonth(TODAY_MONTH)
  }

  const navProps = { year, month, onPrev: prevMonth, onNext: nextMonth, onToday: goToday }

  return (
    <>
      {/* Desktop: full monthly calendar */}
      <main
        className="hidden flex-1 overflow-y-auto bg-canvas p-6 md:flex md:flex-col"
        aria-label="Vista de agenda mensual"
      >
        <CalendarGrid {...navProps} />
      </main>

      {/* Mobile: next 7 days daily view */}
      <main
        className="flex flex-1 flex-col overflow-y-auto bg-canvas px-4 py-5 md:hidden"
        aria-label="Agenda de los próximos 7 días"
      >
        <DailyAgenda {...navProps} />
      </main>
    </>
  )
}
