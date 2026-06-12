import { useEffect, useMemo, useRef, useState } from "react";
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
import { Skeleton } from "@/components/ui/skeleton";
import { Header } from "@/components/taskify/header";
import { KanbanBoard } from "@/components/taskify/kanban-board";
import { MobileTaskList } from "@/components/taskify/mobile-task-list";
import type { CurrentView } from "@/components/taskify/navigation";
import { Sidebar } from "@/components/taskify/sidebar";
import { FinancialControlView } from "@/components/financial-control-view";
import { notifyCriticalAlerts } from "@/lib/notifications";
import { parseTaskDueDate } from "@/lib/task-dates";
import { getFriendlyErrorMessage } from "@/services/api";
import { getBoards } from "@/services/boardService";
import {
  getFinancialSummary,
  getTransactions,
  type FinancialTransaction,
} from "@/services/financial_api";
import { getTasks, type Task } from "@/services/taskService";
import { useAuthStore } from "@/store/useAuthStore";
import { AllTasksView } from "@/components/AllTasksView";

type DashboardAlert = {
  id: string;
  dueDate: Date;
  icon: typeof AlertCircle;
  badge: string;
  badgeVariant: "destructive" | "secondary";
  className: string;
  title: string;
  detail: string;
};

const CRITICAL_ALERTS_NOTIFICATION_SESSION_KEY =
  "taskify-critical-alerts-notified";

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

function isTaskDueToday(task: Task) {
  const dueDate = parseTaskDueDate(task.dueDate);
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

function getCurrentMonthRange() {
  const currentDate = new Date();
  const startDate = new Date(
    currentDate.getFullYear(),
    currentDate.getMonth(),
    1,
  );
  const endDate = new Date(
    currentDate.getFullYear(),
    currentDate.getMonth() + 1,
    0,
  );

  return {
    startDate: formatDateForAPI(startDate),
    endDate: formatDateForAPI(endDate),
    label: new Intl.DateTimeFormat("es-MX", {
      month: "long",
      year: "numeric",
    }).format(startDate),
  };
}

function formatDateForAPI(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");

  return `${year}-${month}-${day}`;
}

function centsToCurrency(cents: number) {
  return cents / 100;
}

function financialPercentage(numerator: number, denominator: number) {
  if (denominator <= 0) {
    return "0.0";
  }

  return ((numerator / denominator) * 100).toFixed(1);
}

function startOfLocalDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function isTaskOverdueOrDueToday(task: Task) {
  if (task.status === "done") {
    return false;
  }

  const dueDate = parseTaskDueDate(task.dueDate);
  if (!dueDate) {
    return false;
  }

  return startOfLocalDay(dueDate).getTime() <= startOfLocalDay(new Date()).getTime();
}

function taskAlertDetail(dueDate: Date) {
  const today = startOfLocalDay(new Date()).getTime();
  const taskDay = startOfLocalDay(dueDate).getTime();
  const timeLabel = new Intl.DateTimeFormat("es-MX", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(dueDate);

  if (taskDay === today) {
    return `Vence hoy a las ${timeLabel}`;
  }

  return `Venció el ${formatAlertDate(dueDate)} · ${timeLabel}`;
}

function paymentAlertDetail(transaction: FinancialTransaction, dueDate: Date) {
  return `Vence el ${formatAlertDate(dueDate)} · ${formatCurrency(
    centsToCurrency(transaction.amountCents),
  )}`;
}

function parseFinancialDate(date: string) {
  const dateOnlyMatch = date.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  const parsedDate = dateOnlyMatch
    ? new Date(
        Number(dateOnlyMatch[1]),
        Number(dateOnlyMatch[2]) - 1,
        Number(dateOnlyMatch[3]),
      )
    : new Date(date);

  return Number.isNaN(parsedDate.getTime()) ? null : parsedDate;
}

function formatAlertDate(date: Date) {
  return new Intl.DateTimeFormat("es-MX", {
    day: "2-digit",
    month: "short",
  }).format(date);
}

export function TaskifyDashboard() {
  const user = useAuthStore((state) => state.user);
  const [currentView, setCurrentView] = useState<CurrentView>("tasks");
  const [selectedBoardId, setSelectedBoardId] = useState<string | null>(null);
  const criticalAlertsNotificationAttemptedRef = useRef(false);
  const currentMonthRange = useMemo(() => getCurrentMonthRange(), []);
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
    isSuccess: globalTasksIsSuccess,
  } = useQuery({
    queryKey: ["tasks", "global"],
    queryFn: () => getTasks(),
    enabled: currentView === "dashboard",
  });
  const {
    data: financialSummary,
    isLoading: financialSummaryLoading,
  } = useQuery({
    queryKey: [
      "financial",
      "summary",
      currentMonthRange.startDate,
      currentMonthRange.endDate,
    ],
    queryFn: () =>
      getFinancialSummary(
        currentMonthRange.startDate,
        currentMonthRange.endDate,
      ),
    enabled: currentView === "dashboard",
  });
  const {
    data: financialTransactions = [],
    isLoading: financialTransactionsLoading,
    isSuccess: financialTransactionsIsSuccess,
  } = useQuery({
    queryKey: [
      "financial",
      "transactions",
      currentMonthRange.startDate,
      currentMonthRange.endDate,
    ],
    queryFn: () =>
      getTransactions({
        startDate: currentMonthRange.startDate,
        endDate: currentMonthRange.endDate,
      }),
    enabled: currentView === "dashboard",
  });

  const taskErrorMessage = getFriendlyErrorMessage(
    error,
    "No se pudo cargar el tablero",
  );
  const boardsErrorMessage = getFriendlyErrorMessage(
    boardsError,
    "No se pudieron cargar los tableros",
  );
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
  const totalIncome = centsToCurrency(financialSummary?.totalIncomeCents ?? 0);
  const totalExpenses = centsToCurrency(
    financialSummary?.totalExpenseCents ?? 0,
  );
  const profitMargin = centsToCurrency(
    financialSummary?.profitMarginCents ?? 0,
  );
  const expensesPercentage = financialPercentage(totalExpenses, totalIncome);
  const marginPercentage = financialPercentage(profitMargin, totalIncome);
  const alerts = useMemo(() => {
    const taskAlerts: DashboardAlert[] = globalTasks
      .filter(isTaskOverdueOrDueToday)
      .flatMap((task) => {
        const dueDate = parseTaskDueDate(task.dueDate);
        if (!dueDate) {
          return [];
        }

        return {
          id: `task-${task.id}`,
          dueDate,
          icon: AlertCircle,
          badge: "Urgente",
          badgeVariant: "destructive" as const,
          className:
            "border-0 bg-[oklab(0.57701_0.217634_0.112472_/_0.1)] text-red-700 hover:bg-[oklab(0.57701_0.217634_0.112472_/_0.16)] dark:text-red-400",
          title: `Tarea por vencer: ${task.title}`,
          detail: taskAlertDetail(dueDate),
        };
      });
    const paymentAlerts: DashboardAlert[] = financialTransactions
      .filter(
        (transaction) =>
          transaction.type === "EXPENSE" && transaction.status === "PENDING",
      )
      .flatMap((transaction) => {
        const dueDate = parseFinancialDate(transaction.date);
        if (!dueDate) {
          return [];
        }

        return {
          id: `payment-${transaction.id}`,
          dueDate,
          icon: Clock,
          badge: "Próximo",
          badgeVariant: "secondary" as const,
          className: "",
          title: `Pago pendiente: ${transaction.concept}`,
          detail: paymentAlertDetail(transaction, dueDate),
        };
      });

    return [...taskAlerts, ...paymentAlerts].sort(
      (firstAlert, secondAlert) =>
        firstAlert.dueDate.getTime() - secondAlert.dueDate.getTime(),
    );
  }, [financialTransactions, globalTasks]);
  const alertsLoading = globalTasksLoading || financialTransactionsLoading;
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

  useEffect(() => {
    const alertsReady =
      currentView === "dashboard" &&
      globalTasksIsSuccess &&
      financialTransactionsIsSuccess &&
      !globalTasksLoading &&
      !financialTransactionsLoading;

    if (!alertsReady || alerts.length === 0) {
      return;
    }

    if (criticalAlertsNotificationAttemptedRef.current) {
      return;
    }
    criticalAlertsNotificationAttemptedRef.current = true;

    try {
      if (sessionStorage.getItem(CRITICAL_ALERTS_NOTIFICATION_SESSION_KEY)) {
        return;
      }

      sessionStorage.setItem(CRITICAL_ALERTS_NOTIFICATION_SESSION_KEY, "true");
      void notifyCriticalAlerts(alerts.length);
    } catch {
      void notifyCriticalAlerts(alerts.length);
    }
  }, [
    alerts.length,
    currentView,
    financialTransactionsIsSuccess,
    financialTransactionsLoading,
    globalTasksIsSuccess,
    globalTasksLoading,
  ]);

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
                      Período: {currentMonthRange.label}
                    </p>
                  </CardHeader>
                  <CardContent className="flex flex-col gap-5 p-6 pt-0">
                    <div className="flex items-center justify-between">
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs uppercase tracking-wide text-muted-foreground">
                          Ingreso total
                        </span>
                        {financialSummaryLoading ? (
                          <Skeleton className="h-8 w-32" />
                        ) : (
                          <span className="text-2xl font-bold text-foreground">
                            {formatCurrency(totalIncome)}
                          </span>
                        )}
                      </div>
                      {financialSummaryLoading ? (
                        <Skeleton className="h-6 w-24 rounded-full" />
                      ) : (
                        <Badge variant="secondary">{marginPercentage}% margen</Badge>
                      )}
                    </div>

                    <div className="my-4 h-px w-full bg-border" />

                    <div className="flex items-center justify-between">
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs uppercase tracking-wide text-muted-foreground">
                          Gasto acumulado
                        </span>
                        {financialSummaryLoading ? (
                          <Skeleton className="h-8 w-32" />
                        ) : (
                          <span className="text-2xl font-bold text-foreground">
                            {formatCurrency(totalExpenses)}
                          </span>
                        )}
                      </div>
                      {financialSummaryLoading ? (
                        <Skeleton className="h-6 w-32 rounded-full" />
                      ) : (
                        <Badge variant="outline">{expensesPercentage}% del ingreso</Badge>
                      )}
                    </div>

                    <div className="my-4 h-px w-full bg-border" />

                    <div className="flex items-center justify-between">
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs uppercase tracking-wide text-muted-foreground">
                          Margen de utilidad
                        </span>
                        {financialSummaryLoading ? (
                          <Skeleton className="h-8 w-32" />
                        ) : (
                          <span className="text-2xl font-bold text-foreground">
                            {formatCurrency(profitMargin)}
                          </span>
                        )}
                      </div>
                      {financialSummaryLoading ? (
                        <Skeleton className="h-6 w-16 rounded-full" />
                      ) : (
                        <Badge>{marginPercentage}%</Badge>
                      )}
                    </div>
                  </CardContent>
                </Card>

                <Card className="border border-border/40 shadow-sm ring-0 lg:col-span-3">
                  <CardHeader className="p-6 pb-2">
                    <CardTitle className="text-base font-semibold">
                      Alertas Críticas
                    </CardTitle>
                    <p className="text-xs text-muted-foreground">
                      {alerts.length} alertas activas
                    </p>
                  </CardHeader>
                  <CardContent className="p-6 pt-0">
                    {alertsLoading ? (
                      <div className="flex flex-col gap-4">
                        {Array.from({ length: 3 }).map((_, index) => (
                          <Skeleton
                            key={index}
                            className="h-16 w-full rounded-lg"
                          />
                        ))}
                      </div>
                    ) : alerts.length === 0 ? (
                      <p className="text-sm text-muted-foreground">
                        Todo al día, no hay alertas urgentes
                      </p>
                    ) : (
                      <ul className="flex flex-col gap-4">
                        {alerts.map(
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
                    )}
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
