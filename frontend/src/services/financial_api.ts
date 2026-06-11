import { apiRequest } from "@/services/api";

export type FinancialTransactionType = "INCOME" | "EXPENSE";
export type FinancialTransactionStatus = "PAID" | "PENDING";

export interface FinancialTransaction {
  id: string;
  type: FinancialTransactionType;
  concept: string;
  category: string;
  amountCents: number;
  date: string;
  status: FinancialTransactionStatus;
  msi?: number | null;
  createdAt: string;
  updatedAt: string;
}

export interface FinancialSummary {
  totalIncomeCents: number;
  totalExpenseCents: number;
  profitMarginCents: number;
}

export interface TransactionDateRange {
  startDate?: string;
  endDate?: string;
}

export interface CreateTransactionInput {
  type: FinancialTransactionType;
  concept: string;
  category: string;
  amountCents: number;
  date: string;
  status: FinancialTransactionStatus;
  msi?: number | null;
}

export type UpdateTransactionInput = CreateTransactionInput;

export async function getTransactions(
  range: TransactionDateRange = {},
): Promise<FinancialTransaction[]> {
  const query = financialDateRangeQuery(range);
  return apiRequest<FinancialTransaction[]>(`/transactions${query}`);
}

export async function getFinancialSummary(
  startDate: string,
  endDate: string,
): Promise<FinancialSummary> {
  const query = financialDateRangeQuery({ startDate, endDate });
  return apiRequest<FinancialSummary>(`/transactions/summary${query}`);
}

export async function createTransaction(
  data: CreateTransactionInput,
): Promise<FinancialTransaction> {
  return apiRequest<FinancialTransaction>("/transactions", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateTransaction(
  id: string,
  data: UpdateTransactionInput,
): Promise<void> {
  await apiRequest<void>(`/transactions/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export async function deleteTransaction(id: string): Promise<void> {
  await apiRequest<void>(`/transactions/${id}`, {
    method: "DELETE",
  });
}

function financialDateRangeQuery(range: TransactionDateRange): string {
  const params = new URLSearchParams();

  if (range.startDate) {
    params.set("start_date", range.startDate);
  }
  if (range.endDate) {
    params.set("end_date", range.endDate);
  }

  const query = params.toString();
  return query ? `?${query}` : "";
}
