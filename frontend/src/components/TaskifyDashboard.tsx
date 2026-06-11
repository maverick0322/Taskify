import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  AlertCircle,
  CheckCircle,
  Clock,
  Layout,
  TrendingUp,
} from "lucide-react";

import { AgendaView } from "@/components/taskify/agenda-view";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Header } from "@/components/taskify/header";
import { KanbanBoard } from "@/components/taskify/kanban-board";
import { MobileTaskList } from "@/components/taskify/mobile-task-list";
import type { CurrentView } from "@/components/taskify/navigation";
import { Sidebar } from "@/components/taskify/sidebar";
import { FinancialControlView } from "@/components/financial-control-view";
import { getBoards } from "@/services/boardService";
import { getTasks, type Task } from "@/services/taskService";
import { useAuthStore } from "@/store/useAuthStore";
import { AllTasksView } from "@/components/AllTasksView";

const FINANCIAL = {
  income: 124_500,
  expenses: 87_320,
  get margin() {
    return this.income - this.expenses;
  },
  get marginPct() {
    return ((this.margin / this.income) * 100).toFixed(1);
  },
};

const ALERTS = [
  {
    id: 1,
    type: "task",
    icon: AlertCircle,
    badge: "Urgente",
    badgeVariant: "destructive" as const,
    className:
      "border-0 bg-[oklab(0.57701_0.217634_0.112472_/_0.1)] text-red-700 hover:bg-[oklab(0.57701_0.217634_0.112472_/_0.16)] dark:text-red-400",
    title: "Tarea por vencer: Informe Q2",
    detail: "Vence hoy a las 18:00 h",
  },
  {
    id: 2,
    type: "payment",
    icon: Clock,
    badge: "Próximo",
    badgeVariant: "secondary" as const,
    className: "",
    title: "Pago a proveedor: Acme Corp.",
    detail: "Vence el 15 jun · $3,200",
  },
];

function formatCurrency(value: number) {
  return value.toLocaleString("es-MX", {
    style: "currency",
    currency: "MXN",
    minimumFractionDigits: 0,
  });
}

function getTodayLabel() {
  return new Date().toLocaleDateString("es-MX", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function parseTaskDate(dueDate: string) {
  if (!dueDate.trim()) {
    return null;
  }

  const dateOnlyMatch = dueDate.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  const date = dateOnlyMatch
    ? new Date(
        Number(dateOnlyMatch[1]),
        Number(dateOnlyMatch[2]) - 1,
        Number(dateOnlyMatch[3]),
      )
    : new Date(dueDate);

  return Number.isNaN(date.getTime()) ? null : date;
}

function isTaskDueToday(task: Task) {
  const dueDate = parseTaskDate(task.dueDate);
  if (!dueDate) {
    return false;
  }

  const today = new Date();
  return (
    dueDate.getFullYear() === today.getFullYear() &&
    dueDate.getMonth() === today.getMonth() &&
    dueDate.getDate() === today.getDate()
  );
}

export function TaskifyDashboard() {
  const user = useAuthStore((state) => state.user);
  const [currentView, setCurrentView] = useState<CurrentView>("tasks");
  const [selectedBoardId, setSelectedBoardId] = useState<string | null>(null);
  const {
    data: boards = [],
    isLoading: boardsLoading,
    isError: boardsIsError,
    error: boardsError,
  } = useQuery({ queryKey: ["boards"], queryFn: getBoards });
  const {
    data: boardTasks = [],
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: ["tasks", selectedBoardId],
    queryFn: () => getTasks(selectedBoardId ?? undefined),
    enabled: !!selectedBoardId,
  });
  const {
    data: globalTasks = [],
    isLoading: globalTasksLoading,
    isError: globalTasksIsError,
  } = useQuery({
    queryKey: ["tasks", "global"],
    queryFn: () => getTasks(),
    enabled: currentView === "dashboard",
  });

  const taskErrorMessage =
    error instanceof Error ? error.message : "No se pudo cargar el tablero";
  const boardsErrorMessage =
    boardsError instanceof Error
      ? boardsError.message
      : "No se pudieron cargar los tableros";
  const selectedBoard = boards.find((board) => board.id === selectedBoardId);
  const todayTaskCount = globalTasks.filter(
    (task) => isTaskDueToday(task) && task.status !== "done",
  ).length;
  const completedTaskCount = globalTasks.filter(
    (task) => task.status === "done",
  ).length;
  const completionRate =
    globalTasks.length === 0
      ? 0
      : Math.round((completedTaskCount / globalTasks.length) * 100);
  const metrics = [
    {
      title: "Tareas para hoy",
      icon: CheckCircle,
      value: globalTasksLoading ? "..." : String(todayTaskCount),
      description: globalTasksIsError
        ? "No se pudieron cargar"
        : `${completedTaskCount} completada${completedTaskCount === 1 ? "" : "s"}`,
    },
    {
      title: "Tasa de finalización",
      icon: TrendingUp,
      value: globalTasksLoading ? "..." : `${completionRate}%`,
      description: globalTasksIsError
        ? "No se pudo calcular"
        : `${completedTaskCount} de ${globalTasks.length} tareas`,
    },
    {
      title: "Tableros activos",
      icon: Layout,
      value: boardsLoading ? "..." : String(boards.length),
      description: boardsIsError
        ? "No se pudieron cargar"
        : `${boards.length} tablero${boards.length === 1 ? "" : "s"}`,
    },
  ];
  const greetingName = user?.firstName || user?.fullName || "Taskify User";

  useEffect(() => {
    if (boards.length === 0) {
      setSelectedBoardId(null);
      return;
    }

    if (!selectedBoardId) {
      return;
    }

    const selectedBoardStillExists = boards.some(
      (board) => board.id === selectedBoardId,
    );

    if (!selectedBoardStillExists) {
      setSelectedBoardId(null);
    }
  }, [boards, selectedBoardId]);

  function handleViewChange(view: CurrentView) {
    if (view === "tasks" || view === "agenda" || view === "financial") {
      setSelectedBoardId(null);
    }

    setCurrentView(view);
  }

  return (
    <div className="flex h-screen w-full overflow-hidden bg-canvas">
      <div className="hidden md:flex md:shrink-0">
        <Sidebar
          className="h-full"
          activeView={currentView}
          boards={boards}
          boardsError={boardsIsError ? boardsErrorMessage : undefined}
          boardsLoading={boardsLoading}
          onViewChange={handleViewChange}
          selectedBoardId={selectedBoardId}
          onBoardSelect={(board) => setSelectedBoardId(board.id)}
        />
      </div>

      <div className="flex flex-1 flex-col overflow-hidden">
        <Header
          activeView={currentView}
          boards={boards}
          boardsError={boardsIsError ? boardsErrorMessage : undefined}
          boardsLoading={boardsLoading}
          onViewChange={handleViewChange}
          selectedBoardId={selectedBoardId}
          selectedBoardName={selectedBoard?.name}
          onBoardSelect={(board) => setSelectedBoardId(board.id)}
        />

        {currentView === "tasks" ? (
          <>
            <div className="flex flex-1 flex-col overflow-hidden md:hidden">
              {selectedBoardId ? (
                <MobileTaskList />
              ) : (
                <AllTasksView />
              )}
            </div>

            <div className="hidden flex-1 overflow-hidden md:flex">
              {!selectedBoardId ? (
                <AllTasksView />
              ) : isLoading ? (
                <div className="flex flex-1 items-center justify-center bg-canvas text-sm font-medium text-muted-foreground">
                  Cargando tablero...
                </div>
              ) : isError ? (
                <div className="flex flex-1 items-center justify-center bg-canvas px-6 text-center text-sm font-medium text-red-600">
                  {taskErrorMessage}
                </div>
              ) : (
                <KanbanBoard selectedBoardId={selectedBoardId} tasks={boardTasks} />
              )}
            </div>
          </>
        ) : null}

        {currentView === "agenda" ? (
          <AgendaView />
        ) : null}

        {currentView === "dashboard" ? (
          <main className="flex-1 overflow-y-auto bg-slate-50 p-6 dark:bg-background md:p-8">
            <div className="mx-auto flex max-w-7xl flex-col gap-8">
              <div className="flex flex-col gap-1">
                <h1 className="text-3xl font-bold tracking-tight text-foreground">
                  Hola, {greetingName}
                </h1>
                <p className="capitalize text-muted-foreground">
                  {getTodayLabel()}
                </p>
              </div>

              <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
                {metrics.map(({ title, icon: Icon, value, description }) => (
                  <Card
                    key={title}
                    className="border border-border/40 shadow-sm ring-0"
                  >
                    <CardHeader className="flex flex-row items-center justify-between p-6 pb-2">
                      <CardTitle className="text-sm font-medium text-muted-foreground">
                        {title}
                      </CardTitle>
                      <Icon className="size-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent className="p-6 pt-0">
                      <p className="text-2xl font-bold text-foreground">
                        {value}
                      </p>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {description}
                      </p>
                    </CardContent>
                  </Card>
                ))}
              </div>

              <div className="grid grid-cols-1 gap-6 lg:grid-cols-7">
                <Card className="border border-border/40 shadow-sm ring-0 lg:col-span-4">
                  <CardHeader className="p-6 pb-2">
                    <CardTitle className="text-base font-semibold">
                      Resumen Financiero Flash
                    </CardTitle>
                    <p className="text-xs text-muted-foreground">
                      Período: junio 2025
                    </p>
                  </CardHeader>
                  <CardContent className="flex flex-col gap-5 p-6 pt-0">
                    <div className="flex items-center justify-between">
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs uppercase tracking-wide text-muted-foreground">
                          Ingreso total
                        </span>
                        <span className="text-2xl font-bold text-foreground">
                          {formatCurrency(FINANCIAL.income)}
                        </span>
                      </div>
                      <Badge variant="secondary">+12% vs mes anterior</Badge>
                    </div>

                    <div className="my-4 h-px w-full bg-border" />

                    <div className="flex items-center justify-between">
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs uppercase tracking-wide text-muted-foreground">
                          Gasto acumulado
                        </span>
                        <span className="text-2xl font-bold text-foreground">
                          {formatCurrency(FINANCIAL.expenses)}
                        </span>
                      </div>
                      <Badge variant="outline">70.1% del ingreso</Badge>
                    </div>

                    <div className="my-4 h-px w-full bg-border" />

                    <div className="flex items-center justify-between">
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs uppercase tracking-wide text-muted-foreground">
                          Margen de utilidad
                        </span>
                        <span className="text-2xl font-bold text-foreground">
                          {formatCurrency(FINANCIAL.margin)}
                        </span>
                      </div>
                      <Badge>{FINANCIAL.marginPct}%</Badge>
                    </div>
                  </CardContent>
                </Card>

                <Card className="border border-border/40 shadow-sm ring-0 lg:col-span-3">
                  <CardHeader className="p-6 pb-2">
                    <CardTitle className="text-base font-semibold">
                      Alertas Críticas
                    </CardTitle>
                    <p className="text-xs text-muted-foreground">
                      {ALERTS.length} alertas activas
                    </p>
                  </CardHeader>
                  <CardContent className="p-6 pt-0">
                    <ul className="flex flex-col gap-4">
                      {ALERTS.map(
                        ({
                          id,
                          icon: Icon,
                          badge,
                          badgeVariant,
                          className,
                          title,
                          detail,
                        }) => (
                          <li
                            key={id}
                            className="flex items-start gap-3 rounded-lg border border-border/40 bg-muted/30 p-4"
                          >
                            <Icon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
                            <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                              <div className="flex flex-wrap items-center gap-2">
                                <Badge
                                  variant={badgeVariant}
                                  className={
                                    className
                                      ? `${className} shrink-0`
                                      : "shrink-0"
                                  }
                                >
                                  {badge}
                                </Badge>
                              </div>
                              <p className="truncate text-sm font-medium text-foreground">
                                {title}
                              </p>
                              <p className="text-xs text-muted-foreground">
                                {detail}
                              </p>
                            </div>
                          </li>
                        ),
                      )}
                    </ul>
                  </CardContent>
                </Card>
              </div>
            </div>
          </main>
        ) : null}

        {currentView === "financial" ? (
          <FinancialControlView />
        ) : null}
      </div>
    </div>
  );
}
