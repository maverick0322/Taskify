const DATE_ONLY_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/;

export function parseTaskDueDate(dueDate: string): Date | null {
  if (!dueDate.trim()) {
    return null;
  }

  const dateOnlyMatch = dueDate.match(DATE_ONLY_PATTERN);
  const date = dateOnlyMatch
    ? new Date(
        Number(dateOnlyMatch[1]),
        Number(dateOnlyMatch[2]) - 1,
        Number(dateOnlyMatch[3]),
      )
    : new Date(dueDate);

  return Number.isNaN(date.getTime()) ? null : date;
}

export function formatTaskDueDateLabel(dueDate: string): string {
  const parsedDate = parseTaskDueDate(dueDate);
  if (!parsedDate) {
    return "Sin fecha";
  }

  const formatterOptions: Intl.DateTimeFormatOptions = {
    day: "2-digit",
    month: "short",
  };

  if (taskDueDateHasDisplayTime(dueDate, parsedDate)) {
    formatterOptions.hour = "2-digit";
    formatterOptions.minute = "2-digit";
  }

  return parsedDate.toLocaleString("es-MX", formatterOptions);
}

export function formatTaskDueDateTime(dueDate: string): string {
  const parsedDate = parseTaskDueDate(dueDate);
  if (!parsedDate) {
    return "Todo el día";
  }

  if (!taskDueDateHasDisplayTime(dueDate, parsedDate)) {
    return "Todo el día";
  }

  return new Intl.DateTimeFormat("es-MX", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(parsedDate);
}

export function taskDueDateInputTime(dueDate: Date): string {
  if (
    dueDate.getHours() === 0 &&
    dueDate.getMinutes() === 0 &&
    dueDate.getSeconds() === 0 &&
    dueDate.getMilliseconds() === 0
  ) {
    return "";
  }

  return `${String(dueDate.getHours()).padStart(2, "0")}:${String(
    dueDate.getMinutes(),
  ).padStart(2, "0")}`;
}

export function taskDueDateToISOString(date: Date, time: string): string {
  const nextDate = new Date(date);
  const [hours = 0, minutes = 0] = time.split(":").map(Number);

  nextDate.setHours(hours, minutes, 0, 0);

  return nextDate.toISOString();
}

function taskDueDateHasDisplayTime(rawDueDate: string, date: Date): boolean {
  if (DATE_ONLY_PATTERN.test(rawDueDate)) {
    return false;
  }

  return (
    date.getHours() !== 0 ||
    date.getMinutes() !== 0 ||
    date.getSeconds() !== 0 ||
    date.getMilliseconds() !== 0
  );
}
