import { apiRequest } from "@/services/api";

export type FinancialTransactionType = "INCOME" | "EXPENSE" | "DEBT_PAYMENT" | "TRANSFER";
export type FinancialAccountType = "CASH" | "DEBIT_CARD" | "CREDIT_CARD";
export type FinancialTransactionStatus = "PAID" | "PENDING" | "COMPLETED";
export type FinancialTransactionRecurrence =
  | "once"
  | "monthly"
  | "quarterly"
  | "biannual"
  | "annual";

export interface FinancialTransaction {
  id: string;
  type: FinancialTransactionType;
  concept: string;
  category: string;
  amountCents: number;
  date: string;
  status: FinancialTransactionStatus;
  msi?: number | null;
  paymentAccountId?: string | null;
  destinationAccountId?: string | null;
  installmentNumber?: number | null;
  installmentCount?: number | null;
  recurrence: FinancialTransactionRecurrence;
  recurrenceLimit?: number | null;
  lastPaidAt?: string | null;
  paidCycles?: FinancialPaidCycle[];
  createdAt: string;
  updatedAt: string;
}

export interface FinancialPaidCycle {
  dueDate: string;
  paidAt: string;
}

export interface FinancialSummary {
  totalIncomeCents: number;
  totalExpenseCents: number;
  profitMarginCents: number;
}

export interface FinancialAccountSummary {
  accountId?: string;
  currentBalanceCents?: number;
  openingBalanceCents?: number;
  creditLimitCents?: number | null;
  currentDebtCents?: number;
  availableCreditCents?: number;
  totalIncomeCents?: number;
  totalExpenseCents?: number;
}

export interface CreditCardSummary {
  id: string;
  name: string;
  bank: string;
  last4: string;
  cutoffDay: number;
  paymentDay: number;
  limitCents: number;
  color: string;
  currentDebtCents: number;
  createdAt: string;
  updatedAt: string;
}

export interface FinancialAccount {
  id: string;
  type: FinancialAccountType;
  name: string;
  institution: string;
  last4?: string | null;
  openingBalanceCents: number;
  currentBalanceCents: number;
  creditLimitCents?: number | null;
  cutoffDay?: number | null;
  paymentDay?: number | null;
  color: string;
  createdAt: string;
  updatedAt: string;
}

export type CreateFinancialAccountInput = Omit<
  FinancialAccount,
  "id" | "currentBalanceCents" | "createdAt" | "updatedAt"
>;

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
  paymentAccountId?: string | null;
  recurrence?: FinancialTransactionRecurrence;
  recurrenceLimit?: number | null;
}

export type UpdateTransactionInput = CreateTransactionInput;

export interface CreateCreditCardInput {
  name: string;
  bank: string;
  last4: string;
  cutoffDay: number;
  paymentDay: number;
  limitCents: number;
  color: string;
}

export interface PayCreditCardDebtInput {
  sourceAccountId: string;
  amountCents: number;
}

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
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function updateAccountPayable(
  id: string,
  data: UpdateTransactionInput,
): Promise<void> {
  await apiRequest<void>(`/accounts-payable/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function payAccountPayable(
  id: string,
  dueDate?: string,
): Promise<void> {
  await apiRequest<void>(`/accounts-payable/${id}/pay`, {
    method: "POST",
    body: dueDate ? JSON.stringify({ dueDate }) : undefined,
  });
}

export async function deleteTransaction(id: string): Promise<void> {
  await apiRequest<void>(`/transactions/${id}`, {
    method: "DELETE",
  });
}

export async function getCreditCards(): Promise<CreditCardSummary[]> {
  return apiRequest<CreditCardSummary[]>("/credit-cards");
}

export async function getFinancialAccounts(): Promise<FinancialAccount[]> {
  return apiRequest<FinancialAccount[]>("/financial-accounts");
}

export async function getFinancialAccountSummary(
  id: string,
): Promise<FinancialAccountSummary> {
  return apiRequest<FinancialAccountSummary>(`/financial-accounts/${id}/summary`);
}

export async function getFinancialAccountTransactions(
  id: string,
  range: TransactionDateRange = {},
): Promise<FinancialTransaction[]> {
  const query = financialDateRangeQuery(range);
  return apiRequest<FinancialTransaction[]>(
    `/financial-accounts/${id}/transactions${query}`,
  );
}

export async function createFinancialAccount(
  data: CreateFinancialAccountInput,
): Promise<FinancialAccount> {
  return apiRequest<FinancialAccount>("/financial-accounts", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateFinancialAccount(
  id: string,
  data: CreateFinancialAccountInput,
): Promise<void> {
  await apiRequest<void>(`/financial-accounts/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteFinancialAccount(id: string): Promise<void> {
  await apiRequest<void>(`/financial-accounts/${id}`, {
    method: "DELETE",
  });
}

export async function createCreditCard(
  data: CreateCreditCardInput,
): Promise<CreditCardSummary> {
  return apiRequest<CreditCardSummary>("/credit-cards", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateCreditCard(
  id: string,
  data: CreateCreditCardInput,
): Promise<void> {
  await apiRequest<void>(`/credit-cards/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export async function payCreditCardDebt(
  id: string,
  data: PayCreditCardDebtInput,
): Promise<void> {
  await apiRequest<void>(`/credit-cards/${id}/pay`, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteCreditCard(id: string): Promise<void> {
  await apiRequest<void>(`/credit-cards/${id}`, {
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
