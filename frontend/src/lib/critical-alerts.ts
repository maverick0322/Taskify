import { AlertCircle, Clock } from "lucide-react";

import { parseTaskDueDate } from "@/lib/task-dates";
import type { FinancialTransaction } from "@/services/financial_api";
import type { Task } from "@/services/taskService";

export type CriticalAlert = {
  id: string;
  type: "task" | "payment";
  dueDate: Date;
  icon: typeof AlertCircle;
  badge: string;
  badgeVariant: "destructive" | "secondary";
  className: string;
  title: string;
  detail: string;
  boardId?: string | null;
  taskId?: string;
};

export function buildCriticalAlerts({
  tasks,
  transactions,
  monthDate = new Date(),
}: {
  tasks: Task[];
  transactions: FinancialTransaction[];
  monthDate?: Date;
}): CriticalAlert[] {
  const taskAlerts: CriticalAlert[] = tasks
    .filter((task) => isTaskOverdueOrDueTodayInMonth(task, monthDate))
    .flatMap((task) => {
      const dueDate = parseTaskDueDate(task.dueDate);
      if (!dueDate) {
        return [];
      }

      return {
        id: `task-${task.id}`,
        type: "task" as const,
        dueDate,
        icon: AlertCircle,
        badge: "Urgente",
        badgeVariant: "destructive" as const,
        className:
          "border-0 bg-[oklab(0.57701_0.217634_0.112472_/_0.1)] text-red-700 hover:bg-[oklab(0.57701_0.217634_0.112472_/_0.16)] dark:text-red-400",
        title: `Tarea por vencer: ${task.title}`,
        detail: taskAlertDetail(dueDate),
        boardId: task.boardId ?? null,
        taskId: task.id,
      };
    });
  const paymentAlerts: CriticalAlert[] = transactions
    .filter(
      (transaction) =>
        transaction.type === "EXPENSE" &&
        transaction.status === "PENDING" &&
        isDateInMonth(parseFinancialDate(transaction.date), monthDate),
    )
    .flatMap((transaction) => {
      const dueDate = parseFinancialDate(transaction.date);
      if (!dueDate) {
        return [];
      }

      return {
        id: `payment-${transaction.id}`,
        type: "payment" as const,
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
}

export function getCurrentMonthRange() {
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

function isTaskOverdueOrDueTodayInMonth(task: Task, monthDate: Date) {
  if (task.status === "done") {
    return false;
  }

  const dueDate = parseTaskDueDate(task.dueDate);
  if (!dueDate || !isDateInMonth(dueDate, monthDate)) {
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

function isDateInMonth(date: Date | null, monthDate: Date) {
  return (
    Boolean(date) &&
    date!.getFullYear() === monthDate.getFullYear() &&
    date!.getMonth() === monthDate.getMonth()
  );
}

function formatAlertDate(date: Date) {
  return new Intl.DateTimeFormat("es-MX", {
    day: "2-digit",
    month: "short",
  }).format(date);
}

function formatDateForAPI(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");

  return `${year}-${month}-${day}`;
}

function formatCurrency(value: number) {
  return value.toLocaleString("es-MX", {
    style: "currency",
    currency: "MXN",
    minimumFractionDigits: 0,
  });
}

function centsToCurrency(cents: number) {
  return cents / 100;
}

function startOfLocalDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}
