"use client"

import { useEffect, useMemo, useState, type ElementType } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  AlertCircle,
  ArrowDownRight,
  ArrowUpRight,
  Building2,
  Calendar,
  ChevronLeft,
  ChevronRight,
  CreditCard,
  Eye,
  Pencil,
  Phone,
  Plus,
  PlusCircle,
  Trash2,
  Wallet,
  Wifi,
  Zap,
} from "lucide-react"

import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { CurrencyInput } from "@/components/ui/currency-input"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { getFriendlyErrorMessage } from "@/services/api"
import {
  createCreditCard,
  createFinancialAccount,
  createTransaction,
  deleteCreditCard,
  deleteFinancialAccount,
  deleteTransaction,
  getFinancialAccountSummary,
  getFinancialAccountTransactions,
  getCreditCards,
  getFinancialAccounts,
  getFinancialSummary,
  getTransactions,
  payAccountPayable,
  payCreditCardDebt,
  updateAccountPayable,
  updateCreditCard,
  updateFinancialAccount,
  updateTransaction,
  type CreateFinancialAccountInput,
  type CreateTransactionInput,
  type CreditCardSummary,
  type FinancialAccount,
  type FinancialAccountSummary,
  type FinancialAccountType,
  type FinancialTransaction,
  type FinancialTransactionRecurrence,
  type PayCreditCardDebtInput,
} from "@/services/financial_api"

type TransactionType = "income" | "expense"

type CardAccountFormInput =
  | (CreateFinancialAccountInput & { type: "DEBIT_CARD" })
  | (CreateFinancialAccountInput & {
      type: "CREDIT_CARD"
      creditLimitCents: number
      cutoffDay: number
      paymentDay: number
    })

interface Transaction {
  id: string
  date: string
  concept: string
  category: string
  type: TransactionType
  amount: number
  msi?: string
  paymentAccountId?: string | null
  paymentMethod?: string
}

interface PendingPayment {
  id: string
  type: "manual" | "credit_card"
  transactionId: string
  creditCardId?: string
  service: string
  icon: ElementType
  dueDate: string
  dueDateRaw: string
  visualState: "normal" | "paid" | "overdue"
  amount: number
}

const RECURRENCE_OPTIONS = [
  { value: "once", label: "Solo una vez" },
  { value: "monthly", label: "Mensual" },
  { value: "quarterly", label: "Trimestral" },
  { value: "biannual", label: "Semestral" },
  { value: "annual", label: "Anual" },
]

const CATEGORIES = [
  "Ingresos",
  "Alimentos",
  "Transporte",
  "Vivienda",
  "Tecnologia",
  "Suscripciones",
  "Salud",
  "Entretenimiento",
  "Educacion",
  "Servicios",
  "Otros",
]

const SERVICES_ICONS = [
  { label: "Electricidad", value: "electricity" },
  { label: "Internet", value: "internet" },
  { label: "Telefonia", value: "phone" },
  { label: "Agua", value: "water" },
  { label: "Gas", value: "gas" },
  { label: "Renta", value: "rent" },
  { label: "Predial", value: "tax" },
  { label: "Streaming", value: "streaming" },
  { label: "Seguro", value: "insurance" },
  { label: "Otro", value: "other" },
]

const CARD_GRADIENTS = [
  { label: "Azul marino", gradient: "from-blue-700 to-blue-900", border: "border-blue-600" },
  { label: "Pizarra", gradient: "from-slate-700 to-slate-900", border: "border-slate-600" },
  { label: "Negro", gradient: "from-zinc-800 to-zinc-950", border: "border-zinc-700" },
  { label: "Rojo", gradient: "from-red-800 to-red-950", border: "border-red-700" },
  { label: "Verde", gradient: "from-emerald-700 to-emerald-900", border: "border-emerald-600" },
  { label: "Dorado", gradient: "from-amber-600 to-amber-800", border: "border-amber-500" },
]

const BANKS = [
  "BBVA Bancomer",
  "Citibanamex",
  "HSBC",
  "Santander",
  "Banorte",
  "Scotiabank",
  "Inbursa",
  "Otro",
]

function fmt(value: number) {
  return new Intl.NumberFormat("es-MX", {
    style: "currency",
    currency: "MXN",
    maximumFractionDigits: 0,
  }).format(value)
}

function amountToCents(value: string) {
  return Math.round(Number(value || "0") * 100)
}

function centsToAmount(value: number) {
  return value / 100
}

function currentMonthRange() {
  const currentDate = new Date()
  const startDate = new Date(currentDate.getFullYear(), currentDate.getMonth(), 1)
  const endDate = new Date(currentDate.getFullYear(), currentDate.getMonth() + 1, 0)

  return {
    startDate: formatDateInput(startDate),
    endDate: formatDateInput(endDate),
    label: new Intl.DateTimeFormat("es-MX", {
      month: "long",
      year: "numeric",
    }).format(startDate),
  }
}

function monthRangeFromDate(date: Date) {
  const startDate = new Date(date.getFullYear(), date.getMonth(), 1)
  const endDate = new Date(date.getFullYear(), date.getMonth() + 1, 0)

  return {
    startDate: formatDateInput(startDate),
    endDate: formatDateInput(endDate),
    label: new Intl.DateTimeFormat("es-MX", {
      month: "long",
      year: "numeric",
    }).format(startDate),
  }
}

function billingCycleRange(cutoffDay: number, anchorDate: Date) {
  const currentCutoffDate = billingCycleDate(
    anchorDate.getFullYear(),
    anchorDate.getMonth(),
    cutoffDay,
  )
  const startsCurrentCycle = anchorDate.getTime() >= currentCutoffDate.getTime()
  const startDate = startsCurrentCycle
    ? currentCutoffDate
    : billingCycleDate(anchorDate.getFullYear(), anchorDate.getMonth() - 1, cutoffDay)
  const endDate = startsCurrentCycle
    ? billingCycleDate(anchorDate.getFullYear(), anchorDate.getMonth() + 1, cutoffDay)
    : currentCutoffDate

  return {
    startDate: formatDateInput(startDate),
    endDate: formatDateInput(endDate),
    label: `${formatDisplayDate(formatDateInput(startDate))} - ${formatDisplayDate(
      formatDateInput(endDate),
    )}`,
  }
}

function billingCycleDate(year: number, monthIndex: number, cutoffDay: number) {
  const firstOfMonth = new Date(year, monthIndex, 1)
  const lastDay = new Date(year, monthIndex + 1, 0).getDate()
  const safeCutoffDay = Math.min(cutoffDay, lastDay)

  return new Date(firstOfMonth.getFullYear(), firstOfMonth.getMonth(), safeCutoffDay)
}

function formatDateInput(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, "0")
  const day = String(date.getDate()).padStart(2, "0")

  return `${year}-${month}-${day}`
}

function formatDisplayDate(rawDate: string) {
  const [year, month, day] = rawDate.split("-").map(Number)
  if (!year || !month || !day) {
    return rawDate
  }

  return new Intl.DateTimeFormat("es-MX", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  }).format(new Date(year, month - 1, day))
}

function parseDateInput(rawDate: string) {
  const [year, month, day] = rawDate.split("-").map(Number)
  if (!year || !month || !day) {
    return null
  }

  return new Date(year, month - 1, day)
}

function todayDate() {
  const currentDate = new Date()
  currentDate.setHours(0, 0, 0, 0)
  return currentDate
}

function nextRecurrenceDate(
  date: Date,
  recurrence: FinancialTransactionRecurrence,
) {
  const nextDate = new Date(date)
  switch (recurrence) {
    case "monthly":
      nextDate.setMonth(nextDate.getMonth() + 1)
      return nextDate
    case "quarterly":
      nextDate.setMonth(nextDate.getMonth() + 3)
      return nextDate
    case "biannual":
      nextDate.setMonth(nextDate.getMonth() + 6)
      return nextDate
    case "annual":
      nextDate.setFullYear(nextDate.getFullYear() + 1)
      return nextDate
    default:
      return nextDate
  }
}

function paymentAccountLabel(account?: FinancialAccount) {
  if (!account) {
    return "Sin metodo"
  }
  const suffix = account.last4 ? ` ....${account.last4}` : ""
  const prefix =
    account.type === "CREDIT_CARD"
      ? "Credito"
      : account.type === "DEBIT_CARD"
      ? "Debito"
      : "Efectivo"
  return `${prefix} - ${account.name}${suffix}`
}

function financialAccountFromCreditCard(
  card: CreditCardSummary,
  accounts: FinancialAccount[],
) {
  return (
    accounts.find((account) => account.id === card.id) ?? {
      id: card.id,
      type: "CREDIT_CARD" as FinancialAccountType,
      name: card.name,
      institution: card.bank,
      last4: card.last4,
      openingBalanceCents: 0,
      currentBalanceCents: card.currentDebtCents,
      creditLimitCents: card.limitCents,
      cutoffDay: card.cutoffDay,
      paymentDay: card.paymentDay,
      color: card.color,
      createdAt: card.createdAt,
      updatedAt: card.updatedAt,
    }
  )
}

function mapTransaction(
  transaction: FinancialTransaction,
  accounts: FinancialAccount[],
): Transaction {
  const account = accounts.find(
    (currentAccount) => currentAccount.id === transaction.paymentAccountId,
  )
  return {
    id: transaction.id,
    date: formatDisplayDate(transaction.date),
    concept: transaction.concept,
    category: transaction.category,
    type: transaction.type === "INCOME" ? "income" : "expense",
    amount: centsToAmount(transaction.amountCents),
    msi: transaction.msi ? `${transaction.msi} MSI` : undefined,
    paymentAccountId: transaction.paymentAccountId,
    paymentMethod: paymentAccountLabel(account),
  }
}

function mapPendingPayments(transaction: FinancialTransaction): PendingPayment[] {
  const dueDate = parseDateInput(transaction.date)
  const currentDate = todayDate()
  if (!dueDate) {
    return [
      pendingPaymentFromTransaction(transaction, transaction.id, transaction.date, "normal"),
    ]
  }
  const paidCycleDates = paidCycleDateSet(transaction)
  const activePaidCycle = activePaidCycleDate(transaction, currentDate)

  if (activePaidCycle && activePaidCycle.getTime() < dueDate.getTime()) {
    return [
      pendingPaymentFromTransaction(
        transaction,
        `${transaction.id}-paid-active`,
        formatDateInput(activePaidCycle),
        "paid",
      ),
    ]
  }

  const overduePayments: PendingPayment[] = []
  let overdueDate = dueDate
  let overdueIndex = 0
  while (overdueDate.getTime() < currentDate.getTime()) {
    const overdueDateRaw = formatDateInput(overdueDate)
    if (!paidCycleDates.has(overdueDateRaw)) {
      overduePayments.push(
        pendingPaymentFromTransaction(
          transaction,
          `${transaction.id}-overdue-${overdueIndex}`,
          overdueDateRaw,
          "overdue",
        ),
      )
    }

    if (transaction.recurrence === "once") {
      return overduePayments
    }

    overdueDate = nextRecurrenceDate(overdueDate, transaction.recurrence)
    overdueIndex += 1
  }

  const currentDueDateRaw = formatDateInput(overdueDate)
  const visualState = paidCycleDates.has(currentDueDateRaw) ? "paid" : "normal"

  return [
    ...overduePayments,
    pendingPaymentFromTransaction(
      transaction,
      `${transaction.id}-current`,
      currentDueDateRaw,
      visualState,
    ),
  ]
}

function paidCycleDateSet(transaction: FinancialTransaction) {
  return new Set((transaction.paidCycles ?? []).map((paidCycle) => paidCycle.dueDate))
}

function activePaidCycleDate(
  transaction: FinancialTransaction,
  currentDate: Date,
) {
  return (transaction.paidCycles ?? [])
    .map((paidCycle) => parseDateInput(paidCycle.dueDate))
    .filter((paidCycleDate): paidCycleDate is Date => Boolean(paidCycleDate))
    .filter((paidCycleDate) => paidCycleDate.getTime() >= currentDate.getTime())
    .sort((leftDate, rightDate) => leftDate.getTime() - rightDate.getTime())[0]
}

function pendingPaymentFromTransaction(
  transaction: FinancialTransaction,
  id: string,
  dueDateRaw: string,
  visualState: PendingPayment["visualState"],
): PendingPayment {
  return {
    id,
    type: "manual",
    transactionId: transaction.id,
    service: transaction.concept,
    icon: iconForCategory(transaction.category),
    dueDate: formatDisplayDate(dueDateRaw),
    dueDateRaw,
    visualState,
    amount: centsToAmount(transaction.amountCents),
  }
}

function mapCreditCardPayable(card: CreditCardSummary): PendingPayment | null {
  if (card.currentDebtCents <= 0) {
    return null
  }
  const dueDate = creditCardPaymentDueDate(card)
  const dueDateRaw = formatDateInput(dueDate)
  const currentDate = todayDate()

  return {
    id: `credit-card-${card.id}`,
    type: "credit_card",
    transactionId: "",
    creditCardId: card.id,
    service: card.name,
    icon: CreditCard,
    dueDate: formatDisplayDate(dueDateRaw),
    dueDateRaw,
    visualState: dueDate.getTime() < currentDate.getTime() ? "overdue" : "normal",
    amount: centsToAmount(card.currentDebtCents),
  }
}

function creditCardPaymentDueDate(card: CreditCardSummary) {
  const currentDate = todayDate()
  const cutoffDate = billingCycleDate(
    currentDate.getFullYear(),
    currentDate.getMonth(),
    card.cutoffDay,
  )
  const dueMonthOffset = currentDate.getTime() > cutoffDate.getTime() ? 2 : 1

  return billingCycleDate(
    currentDate.getFullYear(),
    currentDate.getMonth() + dueMonthOffset,
    card.paymentDay,
  )
}

function iconForCategory(category: string): ElementType {
  const normalizedCategory = category.toLowerCase()
  if (normalizedCategory.includes("electric") || normalizedCategory.includes("luz")) {
    return Zap
  }
  if (normalizedCategory.includes("internet")) {
    return Wifi
  }
  if (normalizedCategory.includes("telefon") || normalizedCategory.includes("celular")) {
    return Phone
  }
  if (normalizedCategory.includes("vivienda") || normalizedCategory.includes("predial")) {
    return Building2
  }

  return Wallet
}

function financialQueryKeys(startDate: string, endDate: string) {
  return {
    transactions: ["financial", "transactions", startDate, endDate] as const,
    payables: ["financial", "accounts-payable"] as const,
    summary: ["financial", "summary", startDate, endDate] as const,
    creditCards: ["financial", "credit-cards"] as const,
    accounts: ["financial", "accounts"] as const,
  }
}

function cardVisualForColor(color: string) {
  return (
    CARD_GRADIENTS.find((gradient) => gradient.gradient === color) ??
    CARD_GRADIENTS[0]
  )
}

function cardCutoffLabel(cutoffDay: number) {
  return `Dia ${cutoffDay}`
}

function serviceValueForCategory(category: string) {
  const normalizedCategory = category.toLowerCase()
  return (
    SERVICES_ICONS.find(
      (service) =>
        service.label.toLowerCase() === normalizedCategory ||
        service.value.toLowerCase() === normalizedCategory,
    )?.value ?? "other"
  )
}

function NewMovementDialog({
  open,
  onClose,
  onSubmit,
  isSaving,
  transaction,
  paymentAccounts,
  submitError,
}: {
  open: boolean
  onClose: () => void
  onSubmit: (data: CreateTransactionInput) => void
  isSaving: boolean
  transaction?: FinancialTransaction | null
  paymentAccounts: FinancialAccount[]
  submitError?: string
}) {
  const [tipo, setTipo] = useState("")
  const [categoria, setCategoria] = useState("")
  const [monto, setMonto] = useState("")
  const [concepto, setConcepto] = useState("")
  const [fecha, setFecha] = useState(formatDateInput(new Date()))
  const [paymentAccountId, setPaymentAccountId] = useState("")
  const [msi, setMsi] = useState("1")
  const [errorMessage, setErrorMessage] = useState("")
  const isEditing = Boolean(transaction?.id)
  const selectedPaymentAccount = paymentAccounts.find(
    (account) => account.id === paymentAccountId,
  )

  useEffect(() => {
    if (!open) {
      return
    }

    setTipo(transaction ? (transaction.type === "INCOME" ? "income" : "expense") : "")
    setCategoria(transaction?.category ?? "")
    setMonto(transaction ? String(centsToAmount(transaction.amountCents)) : "")
    setConcepto(transaction?.concept ?? "")
    setFecha(transaction?.date ?? formatDateInput(new Date()))
    setPaymentAccountId(transaction?.paymentAccountId ?? paymentAccounts[0]?.id ?? "")
    setMsi(transaction?.msi ? String(transaction.msi) : "1")
    setErrorMessage("")
  }, [open, paymentAccounts, transaction])

  const handleSubmit = () => {
    if (!concepto.trim() || !tipo || !categoria || !fecha || !paymentAccountId || amountToCents(monto) <= 0) {
      setErrorMessage("Completa concepto, tipo, categoria, metodo de pago, fecha y un monto valido.")
      return
    }
    const msiValue =
      tipo === "expense" && selectedPaymentAccount?.type === "CREDIT_CARD"
        ? Number(msi)
        : null
    if (msiValue !== null && (!Number.isInteger(msiValue) || msiValue < 1)) {
      setErrorMessage("Los MSI deben ser un numero entero mayor a cero.")
      return
    }

    onSubmit({
      type: tipo === "income" ? "INCOME" : "EXPENSE",
      concept: concepto.trim(),
      category: categoria,
      amountCents: amountToCents(monto),
      date: fecha,
      status: "PAID",
      msi: msiValue,
      paymentAccountId,
    })
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Editar Movimiento" : "Nuevo Movimiento"}</DialogTitle>
          <DialogDescription>
            Registra un ingreso o egreso en tu libro financiero.
          </DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-1 gap-4 py-2 sm:grid-cols-2">
          <div className="flex flex-col gap-1.5 sm:col-span-2">
            <Label htmlFor="concepto">Concepto</Label>
            <Input
              id="concepto"
              placeholder="Ej. Sueldo mensual, Netflix..."
              value={concepto}
              onChange={(event) => setConcepto(event.target.value)}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="monto">Monto</Label>
            <CurrencyInput
              id="monto"
              value={monto}
              onValueChange={setMonto}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="fecha-movimiento">Fecha</Label>
            <Input
              id="fecha-movimiento"
              type="date"
              value={fecha}
              onChange={(event) => setFecha(event.target.value)}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Tipo</Label>
            <Select value={tipo} onValueChange={setTipo}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Selecciona" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="income">Ingreso</SelectItem>
                  <SelectItem value="expense">Egreso</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Categoria</Label>
            <Select value={categoria} onValueChange={setCategoria}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Selecciona" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {CATEGORIES.map((category) => (
                    <SelectItem key={category} value={category}>
                      {category}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Metodo de Pago</Label>
            <Select value={paymentAccountId} onValueChange={setPaymentAccountId}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Selecciona" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {paymentAccounts.map((account) => (
                    <SelectItem key={account.id} value={account.id}>
                      {paymentAccountLabel(account)}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          {selectedPaymentAccount?.type === "CREDIT_CARD" && tipo === "expense" ? (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="msi">MSI</Label>
              <Input
                id="msi"
                type="number"
                min={1}
                value={msi}
                onChange={(event) => setMsi(event.target.value)}
              />
            </div>
          ) : null}

          {errorMessage || submitError ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm font-medium text-red-700 sm:col-span-2">
              {errorMessage || submitError}
            </p>
          ) : null}
        </div>

        <DialogFooter showCloseButton>
          <Button
            type="submit"
            className="w-full sm:w-auto"
            disabled={isSaving}
            onClick={handleSubmit}
          >
            {isSaving ? "Guardando..." : "Guardar movimiento"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function NewPaymentDialog({
  open,
  onClose,
  onSubmit,
  isSaving,
  accountPayable,
}: {
  open: boolean
  onClose: () => void
  onSubmit: (data: CreateTransactionInput) => void
  isSaving: boolean
  accountPayable?: FinancialTransaction | null
}) {
  const [categoria, setCategoria] = useState("")
  const [recurrencia, setRecurrencia] =
    useState<FinancialTransactionRecurrence>("monthly")
  const [servicio, setServicio] = useState("")
  const [monto, setMonto] = useState("")
  const [fechaVence, setFechaVence] = useState("")
  const [limiteRecurrencia, setLimiteRecurrencia] = useState("")
  const [errorMessage, setErrorMessage] = useState("")
  const isEditing = Boolean(accountPayable?.id)

  useEffect(() => {
    if (!open) {
      return
    }

    setCategoria(accountPayable ? serviceValueForCategory(accountPayable.category) : "")
    setRecurrencia("monthly")
    setServicio(accountPayable?.concept ?? "")
    setMonto(accountPayable ? String(centsToAmount(accountPayable.amountCents)) : "")
    setFechaVence(accountPayable?.date ?? "")
    setLimiteRecurrencia(
      accountPayable?.recurrenceLimit ? String(accountPayable.recurrenceLimit) : "",
    )
    setErrorMessage("")
  }, [accountPayable, open])

  const handleSubmit = () => {
    const selectedService = SERVICES_ICONS.find((service) => service.value === categoria)
    const recurrenceLimit =
      recurrencia === "once" || limiteRecurrencia.trim() === ""
        ? null
        : Number(limiteRecurrencia)
    if (!servicio.trim() || !categoria || !fechaVence || amountToCents(monto) <= 0) {
      setErrorMessage("Completa servicio, categoria, fecha y un monto valido.")
      return
    }
    if (
      recurrenceLimit !== null &&
      (!Number.isInteger(recurrenceLimit) || recurrenceLimit <= 0)
    ) {
      setErrorMessage("El limite de pagos debe ser un numero entero mayor a cero.")
      return
    }

    onSubmit({
      type: "EXPENSE",
      concept: servicio.trim(),
      category: selectedService?.label ?? categoria,
      amountCents: amountToCents(monto),
      date: fechaVence,
      status: "PENDING",
      msi: null,
      recurrence: recurrencia,
      recurrenceLimit,
    })
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? "Editar Cuenta por Pagar" : "Nueva Cuenta por Pagar"}
          </DialogTitle>
          <DialogDescription>
            Registra un servicio o pago recurrente pendiente.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-5 py-2">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="servicio">Servicio / Concepto</Label>
            <Input
              id="servicio"
              placeholder="Ej. CFE - Luz, Predial..."
              value={servicio}
              onChange={(event) => setServicio(event.target.value)}
            />
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="monto-pago">Monto</Label>
              <CurrencyInput
                id="monto-pago"
                value={monto}
                onValueChange={setMonto}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="fecha-vence">Fecha de vencimiento</Label>
              <Input
                id="fecha-vence"
                type="date"
                value={fechaVence}
                onChange={(event) => setFechaVence(event.target.value)}
              />
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Categoria</Label>
            <Select value={categoria} onValueChange={setCategoria}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Selecciona categoria" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {SERVICES_ICONS.map((service) => (
                    <SelectItem key={service.value} value={service.value}>
                      {service.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-3">
            <Label>Recurrencia</Label>
            <RadioGroup
              value={recurrencia}
              onValueChange={(value) =>
                setRecurrencia(value as FinancialTransactionRecurrence)
              }
              className="grid grid-cols-1 gap-2"
            >
              {RECURRENCE_OPTIONS.map((option) => (
                <label
                  key={option.value}
                  className={cn(
                    "flex cursor-pointer items-center gap-3 rounded-lg border px-4 py-3 transition-colors",
                    recurrencia === option.value
                      ? "border-primary/50 bg-primary/5"
                      : "border-border hover:bg-muted/50",
                  )}
                >
                  <RadioGroupItem value={option.value} id={`pay-${option.value}`} />
                  <span className="text-sm font-medium text-foreground">
                    {option.label}
                  </span>
                </label>
              ))}
            </RadioGroup>
          </div>
          {recurrencia !== "once" ? (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="limite-recurrencia">Limite de pagos</Label>
              <Input
                id="limite-recurrencia"
                type="number"
                min={1}
                step={1}
                placeholder="Indefinido"
                value={limiteRecurrencia}
                onChange={(event) => setLimiteRecurrencia(event.target.value)}
              />
            </div>
          ) : null}
          {errorMessage ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm font-medium text-red-700">
              {errorMessage}
            </p>
          ) : null}
        </div>

        <DialogFooter showCloseButton>
          <Button
            type="submit"
            className="w-full sm:w-auto"
            disabled={isSaving}
            onClick={handleSubmit}
          >
            {isSaving ? "Guardando..." : "Guardar pago"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function AddCardDialog({
  open,
  onClose,
  onSubmit,
  isSaving,
  card,
  debitCard,
}: {
  open: boolean
  onClose: () => void
  onSubmit: (data: CardAccountFormInput) => void
  isSaving: boolean
  card?: CreditCardSummary | null
  debitCard?: FinancialAccount | null
}) {
  const [cardType, setCardType] = useState<Exclude<FinancialAccountType, "CASH">>(
    "DEBIT_CARD",
  )
  const [name, setName] = useState("")
  const [bank, setBank] = useState("")
  const [last4, setLast4] = useState("")
  const [cutoffDay, setCutoffDay] = useState("")
  const [paymentDay, setPaymentDay] = useState("")
  const [limit, setLimit] = useState("")
  const [openingBalance, setOpeningBalance] = useState("")
  const [selectedGradient, setSelectedGradient] = useState(CARD_GRADIENTS[0])
  const [errorMessage, setErrorMessage] = useState("")
  const isEditing = Boolean(card?.id || debitCard?.id)

  useEffect(() => {
    if (!open) {
      return
    }

    setCardType(card ? "CREDIT_CARD" : "DEBIT_CARD")
    setName(card?.name ?? debitCard?.name ?? "")
    setBank(card?.bank ?? debitCard?.institution ?? "")
    setLast4(card?.last4 ?? debitCard?.last4 ?? "")
    setCutoffDay(card ? String(card.cutoffDay) : "")
    setPaymentDay(card ? String(card.paymentDay) : "")
    setLimit(card ? String(centsToAmount(card.limitCents)) : "")
    setOpeningBalance(
      debitCard ? String(centsToAmount(debitCard.openingBalanceCents)) : "",
    )
    setSelectedGradient(
      card
        ? cardVisualForColor(card.color)
        : debitCard
        ? cardVisualForColor(debitCard.color)
        : CARD_GRADIENTS[0],
    )
    setErrorMessage("")
  }, [card, debitCard, open])

  const handleSubmit = () => {
    const trimmedName = name.trim()
    if (!trimmedName || !bank || last4.length !== 4) {
      setErrorMessage("Completa nombre, banco y los ultimos 4 digitos.")
      return
    }
    if (cardType === "DEBIT_CARD") {
      const openingBalanceCents = amountToCents(openingBalance)
      if (openingBalanceCents < 0) {
        setErrorMessage("El saldo inicial no puede ser negativo.")
        return
      }
      onSubmit({
        type: "DEBIT_CARD",
        name: trimmedName,
        institution: bank,
        last4,
        openingBalanceCents,
        creditLimitCents: null,
        cutoffDay: null,
        paymentDay: null,
        color: selectedGradient.gradient,
      })
      return
    }

    const parsedCutoffDay = Number(cutoffDay)
    const parsedPaymentDay = Number(paymentDay)
    const creditLimitCents = amountToCents(limit)
    if (
      creditLimitCents <= 0 ||
      parsedCutoffDay < 1 ||
      parsedCutoffDay > 31 ||
      parsedPaymentDay < 1 ||
      parsedPaymentDay > 31
    ) {
      setErrorMessage("Completa limite, dia de corte y dia limite de pago validos.")
      return
    }
    onSubmit({
      type: "CREDIT_CARD",
      name: trimmedName,
      institution: bank,
      last4,
      openingBalanceCents: 0,
      creditLimitCents,
      cutoffDay: parsedCutoffDay,
      paymentDay: parsedPaymentDay,
      color: selectedGradient.gradient,
    })
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Editar Tarjeta" : "Agregar Tarjeta"}</DialogTitle>
          <DialogDescription>
            Registra una tarjeta de debito o credito para tus movimientos.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-5 py-2">
          <div className="flex flex-col gap-3">
            <Label>Tipo de Tarjeta</Label>
            <RadioGroup
              value={cardType}
              onValueChange={(value) => {
                setCardType(value as Exclude<FinancialAccountType, "CASH">)
                setErrorMessage("")
              }}
              className="grid grid-cols-2 gap-2"
            >
              <label
                className={cn(
                  "flex cursor-pointer items-center gap-3 rounded-lg border px-4 py-3 transition-colors",
                  cardType === "DEBIT_CARD"
                    ? "border-primary/50 bg-primary/5"
                    : "border-border hover:bg-muted/50",
                )}
              >
                <RadioGroupItem
                  value="DEBIT_CARD"
                  id="card-type-debit"
                  disabled={isEditing}
                />
                <span className="text-sm font-medium text-foreground">Debito</span>
              </label>
              <label
                className={cn(
                  "flex cursor-pointer items-center gap-3 rounded-lg border px-4 py-3 transition-colors",
                  cardType === "CREDIT_CARD"
                    ? "border-primary/50 bg-primary/5"
                    : "border-border hover:bg-muted/50",
                )}
              >
                <RadioGroupItem
                  value="CREDIT_CARD"
                  id="card-type-credit"
                  disabled={isEditing}
                />
                <span className="text-sm font-medium text-foreground">Credito</span>
              </label>
            </RadioGroup>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="card-name">Nombre de la tarjeta</Label>
            <Input
              id="card-name"
              placeholder="Ej. Clasica, Oro, LikeU..."
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Banco emisor</Label>
            <Select value={bank} onValueChange={setBank}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Selecciona banco" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {BANKS.map((bankName) => (
                    <SelectItem key={bankName} value={bankName}>
                      {bankName}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="last-four">Ultimos 4 digitos</Label>
              <Input
                id="last-four"
                placeholder="0000"
                maxLength={4}
                className="font-mono tracking-widest"
                value={last4}
                onChange={(event) =>
                  setLast4(event.target.value.replace(/\D/g, "").slice(0, 4))
                }
              />
            </div>
            {cardType === "DEBIT_CARD" ? (
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="opening-balance">Saldo inicial</Label>
                <CurrencyInput
                  id="opening-balance"
                  value={openingBalance}
                  onValueChange={setOpeningBalance}
                />
              </div>
            ) : (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="cutoff-day">Dia de corte</Label>
              <Input
                id="cutoff-day"
                type="number"
                min={1}
                max={31}
                placeholder="15"
                value={cutoffDay}
                onChange={(event) => setCutoffDay(event.target.value)}
              />
            </div>
            )}
          </div>

          {cardType === "CREDIT_CARD" ? (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="payment-day">Dia limite de pago</Label>
              <Input
                id="payment-day"
                type="number"
                min={1}
                max={31}
                placeholder="5"
                value={paymentDay}
                onChange={(event) => setPaymentDay(event.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="limit">Limite de credito</Label>
              <CurrencyInput
                id="limit"
                value={limit}
                onValueChange={setLimit}
              />
            </div>
            </div>
          ) : null}

          <div className="flex flex-col gap-2">
            <Label>Color del plastico</Label>
            <div className="flex flex-wrap gap-2">
              {CARD_GRADIENTS.map((gradient) => (
                <button
                  key={gradient.gradient}
                  type="button"
                  title={gradient.label}
                  onClick={() => setSelectedGradient(gradient)}
                  className={cn(
                    "size-8 cursor-pointer rounded-full bg-gradient-to-br ring-2 ring-offset-2 transition-all",
                    gradient.gradient,
                    selectedGradient.gradient === gradient.gradient
                      ? "ring-primary"
                      : "ring-transparent hover:ring-border",
                  )}
                />
              ))}
            </div>
          </div>
          {errorMessage ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm font-medium text-red-700">
              {errorMessage}
            </p>
          ) : null}
        </div>

        <DialogFooter showCloseButton>
          <Button
            type="submit"
            className="w-full sm:w-auto"
            disabled={isSaving}
            onClick={handleSubmit}
          >
            {isSaving ? "Guardando..." : isEditing ? "Guardar cambios" : "Agregar tarjeta"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function AccountDetailModal({
  account,
  open,
  onClose,
}: {
  account: FinancialAccount | null
  open: boolean
  onClose: () => void
}) {
  const [periodAnchor, setPeriodAnchor] = useState(new Date())

  useEffect(() => {
    if (open) {
      setPeriodAnchor(new Date())
    }
  }, [account?.id, open])

  const period = useMemo(() => {
    if (!account || account.type !== "CREDIT_CARD") {
      return monthRangeFromDate(periodAnchor)
    }
    return billingCycleRange(account.cutoffDay ?? 1, periodAnchor)
  }, [account, periodAnchor])

  const summaryQuery = useQuery({
    queryKey: ["financial", "accounts", account?.id, "summary"],
    queryFn: () => getFinancialAccountSummary(account?.id ?? ""),
    enabled: open && Boolean(account?.id),
  })
  const transactionsQuery = useQuery({
    queryKey: [
      "financial",
      "accounts",
      account?.id,
      "transactions",
      period.startDate,
      period.endDate,
    ],
    queryFn: () =>
      getFinancialAccountTransactions(account?.id ?? "", {
        startDate: period.startDate,
        endDate: period.endDate,
      }),
    enabled: open && Boolean(account?.id),
  })

  if (!account) {
    return null
  }

  const summary = summaryQuery.data
  const creditLimitCents =
    summaryValue(summary, "creditLimitCents") ?? account.creditLimitCents ?? 0
  const currentDebtCents =
    summaryValue(summary, "currentDebtCents") ??
    (account.type === "CREDIT_CARD" ? account.currentBalanceCents : 0)
  const availableCreditCents =
    summaryValue(summary, "availableCreditCents") ??
    Math.max(creditLimitCents - currentDebtCents, 0)
  const currentBalanceCents =
    summaryValue(summary, "currentBalanceCents") ?? account.currentBalanceCents
  const creditUsage =
    creditLimitCents > 0
      ? Math.min((currentDebtCents / creditLimitCents) * 100, 100)
      : 0
  const transactions = transactionsQuery.data ?? []
  const isCreditAccount = account.type === "CREDIT_CARD"

  function movePeriod(offset: number) {
    setPeriodAnchor(
      (currentDate) =>
        new Date(
          currentDate.getFullYear(),
          currentDate.getMonth() + offset,
          currentDate.getDate(),
        ),
    )
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{account.name}</DialogTitle>
          <DialogDescription>
            {isCreditAccount ? "Detalle de tarjeta de credito" : "Detalle de debito"}
            {account.last4 ? ` - terminacion ${account.last4}` : ""}
          </DialogDescription>
        </DialogHeader>

        {isCreditAccount ? (
          <div className="grid gap-4 sm:grid-cols-3">
            <div className="rounded-lg border bg-card p-4">
              <span className="text-xs font-medium uppercase text-muted-foreground">
                Limite de credito
              </span>
              <p className="mt-2 text-xl font-semibold">
                {fmt(centsToAmount(creditLimitCents))}
              </p>
            </div>
            <div className="rounded-lg border bg-card p-4">
              <span className="text-xs font-medium uppercase text-muted-foreground">
                Deuda actual
              </span>
              <p className="mt-2 text-xl font-semibold text-red-600">
                {fmt(centsToAmount(currentDebtCents))}
              </p>
            </div>
            <div className="rounded-lg border bg-card p-4">
              <span className="text-xs font-medium uppercase text-muted-foreground">
                Credito disponible
              </span>
              <p className="mt-2 text-xl font-semibold text-emerald-600">
                {fmt(centsToAmount(availableCreditCents))}
              </p>
            </div>
            <div className="sm:col-span-3 rounded-lg border bg-muted/30 p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex flex-col gap-1">
                  <span className="text-sm font-medium">Uso de linea</span>
                  <span className="text-xs text-muted-foreground">
                    Corte dia {account.cutoffDay ?? "-"} · Pago dia{" "}
                    {account.paymentDay ?? "-"}
                  </span>
                </div>
                <span className="text-sm font-semibold">
                  {creditUsage.toFixed(0)}%
                </span>
              </div>
              <div className="mt-3 h-2 overflow-hidden rounded-full bg-background">
                <div
                  className="h-full rounded-full bg-primary transition-all"
                  style={{ width: `${creditUsage}%` }}
                />
              </div>
            </div>
          </div>
        ) : (
          <div className="rounded-lg border bg-card p-5">
            <span className="text-xs font-medium uppercase text-muted-foreground">
              Saldo actual
            </span>
            <p className="mt-2 text-3xl font-bold">
              {fmt(centsToAmount(currentBalanceCents))}
            </p>
            <p className="mt-1 text-sm text-muted-foreground">
              {account.institution || "Cuenta de debito"}
            </p>
          </div>
        )}

        <div className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h3 className="text-sm font-semibold">
                {isCreditAccount ? "Movimientos del ciclo" : "Movimientos del mes"}
              </h3>
              <p className="text-xs text-muted-foreground">{period.label}</p>
            </div>
            <div className="flex items-center gap-1">
              <Button
                type="button"
                variant="outline"
                size="icon-sm"
                aria-label="Periodo anterior"
                onClick={() => movePeriod(-1)}
              >
                <ChevronLeft className="size-4" />
              </Button>
              <Button
                type="button"
                variant="outline"
                size="icon-sm"
                aria-label="Periodo siguiente"
                onClick={() => movePeriod(1)}
              >
                <ChevronRight className="size-4" />
              </Button>
            </div>
          </div>

          <div className="overflow-hidden rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead>Fecha</TableHead>
                  <TableHead>Concepto</TableHead>
                  <TableHead className="hidden md:table-cell">Categoria</TableHead>
                  <TableHead className="hidden md:table-cell">Tipo</TableHead>
                  <TableHead className="text-right">Monto</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {transactionsQuery.isLoading ? (
                  Array.from({ length: 4 }).map((_, index) => (
                    <TableRow key={index}>
                      <TableCell>
                        <Skeleton className="h-4 w-20" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-4 w-full" />
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <Skeleton className="h-4 w-24" />
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <Skeleton className="h-4 w-20" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="ml-auto h-4 w-20" />
                      </TableCell>
                    </TableRow>
                  ))
                ) : transactions.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className="h-24 text-center text-sm text-muted-foreground"
                    >
                      No hay movimientos en este periodo.
                    </TableCell>
                  </TableRow>
                ) : (
                  transactions.map((transaction) => (
                    <TableRow key={transaction.id}>
                      <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                        {formatDisplayDate(transaction.date)}
                      </TableCell>
                      <TableCell className="font-medium">
                        <span>{transaction.concept}</span>
                        {transaction.installmentNumber &&
                        transaction.installmentCount ? (
                          <Badge
                            variant="secondary"
                            className="ml-2 text-xs font-normal"
                          >
                            {transaction.installmentNumber}/
                            {transaction.installmentCount}
                          </Badge>
                        ) : transaction.msi ? (
                          <Badge
                            variant="secondary"
                            className="ml-2 text-xs font-normal"
                          >
                            {transaction.msi} MSI
                          </Badge>
                        ) : null}
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <Badge variant="outline" className="font-normal">
                          {transaction.category}
                        </Badge>
                      </TableCell>
                      <TableCell className="hidden md:table-cell text-muted-foreground">
                        {transaction.type === "INCOME"
                          ? "Ingreso"
                          : transaction.type === "DEBT_PAYMENT"
                          ? "Pago de deuda"
                          : transaction.type === "TRANSFER"
                          ? "Transferencia"
                          : "Egreso"}
                      </TableCell>
                      <TableCell
                        className={cn(
                          "text-right font-semibold tabular-nums",
                          transaction.type === "INCOME"
                            ? "text-emerald-600"
                            : "text-foreground",
                        )}
                      >
                        {transaction.type === "INCOME" ? "+" : "-"}
                        {fmt(centsToAmount(transaction.amountCents))}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {summaryQuery.isError || transactionsQuery.isError ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm font-medium text-red-700">
              No se pudo cargar el detalle de la cuenta.
            </p>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  )
}

function summaryValue(
  summary: FinancialAccountSummary | undefined,
  key: keyof FinancialAccountSummary,
) {
  const value = summary?.[key]
  return typeof value === "number" ? value : undefined
}

function PayCreditCardDialog({
  open,
  onClose,
  card,
  sourceAccounts,
  onSubmit,
  isSaving,
}: {
  open: boolean
  onClose: () => void
  card: CreditCardSummary | null
  sourceAccounts: FinancialAccount[]
  onSubmit: (cardId: string, data: PayCreditCardDebtInput) => void
  isSaving: boolean
}) {
  const [sourceAccountId, setSourceAccountId] = useState("")
  const [errorMessage, setErrorMessage] = useState("")

  useEffect(() => {
    if (!open) {
      return
    }
    setSourceAccountId(sourceAccounts[0]?.id ?? "")
    setErrorMessage("")
  }, [open, sourceAccounts])

  if (!card) {
    return null
  }

  const activeCard = card
  const selectedSource = sourceAccounts.find(
    (account) => account.id === sourceAccountId,
  )

  function handleSubmit() {
    if (!sourceAccountId) {
      setErrorMessage("Selecciona la cuenta de origen del pago.")
      return
    }
    if (
      selectedSource &&
      selectedSource.type !== "CASH" &&
      selectedSource.currentBalanceCents < activeCard.currentDebtCents
    ) {
      setErrorMessage("La cuenta seleccionada no tiene saldo suficiente.")
      return
    }
    onSubmit(activeCard.id, {
      sourceAccountId,
      amountCents: activeCard.currentDebtCents,
    })
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Pagar tarjeta</DialogTitle>
          <DialogDescription>
            Liquida la deuda de {card.name} sin duplicar tus egresos.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4 py-2">
          <div className="rounded-lg border bg-muted/30 p-4">
            <span className="text-xs font-medium uppercase text-muted-foreground">
              Monto a pagar
            </span>
            <p className="mt-1 text-2xl font-bold">
              {fmt(centsToAmount(card.currentDebtCents))}
            </p>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Cuenta de origen</Label>
            <Select value={sourceAccountId} onValueChange={setSourceAccountId}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Selecciona" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {sourceAccounts.map((account) => (
                    <SelectItem key={account.id} value={account.id}>
                      {paymentAccountLabel(account)} ·{" "}
                      {fmt(centsToAmount(account.currentBalanceCents))}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          {errorMessage ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm font-medium text-red-700">
              {errorMessage}
            </p>
          ) : null}
        </div>

        <DialogFooter showCloseButton>
          <Button
            type="submit"
            className="w-full sm:w-auto"
            disabled={isSaving}
            onClick={handleSubmit}
          >
            {isSaving ? "Pagando..." : "Pagar tarjeta"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function FinancialControlView() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [paymentDialogOpen, setPaymentDialogOpen] = useState(false)
  const [addCardDialogOpen, setAddCardDialogOpen] = useState(false)
  const [transactionToDelete, setTransactionToDelete] =
    useState<FinancialTransaction | null>(null)
  const [transactionToEdit, setTransactionToEdit] =
    useState<FinancialTransaction | null>(null)
  const [accountPayableToEdit, setAccountPayableToEdit] =
    useState<FinancialTransaction | null>(null)
  const [creditCardToEdit, setCreditCardToEdit] =
    useState<CreditCardSummary | null>(null)
  const [creditCardToDelete, setCreditCardToDelete] =
    useState<CreditCardSummary | null>(null)
  const [debitCardToEdit, setDebitCardToEdit] =
    useState<FinancialAccount | null>(null)
  const [debitCardToDelete, setDebitCardToDelete] =
    useState<FinancialAccount | null>(null)
  const [accountDetail, setAccountDetail] = useState<FinancialAccount | null>(null)
  const [creditCardToPay, setCreditCardToPay] =
    useState<CreditCardSummary | null>(null)
  const [transactionDialogError, setTransactionDialogError] = useState("")
  const queryClient = useQueryClient()
  const monthRange = useMemo(() => currentMonthRange(), [])
  const queryKeys = useMemo(
    () => financialQueryKeys(monthRange.startDate, monthRange.endDate),
    [monthRange.endDate, monthRange.startDate],
  )

  const {
    data: apiTransactions = [],
    isLoading: isTransactionsLoading,
  } = useQuery({
    queryKey: queryKeys.transactions,
    queryFn: () =>
      getTransactions({
        startDate: monthRange.startDate,
        endDate: monthRange.endDate,
      }),
  })
  const {
    data: apiPayables = [],
    isLoading: isPayablesLoading,
  } = useQuery({
    queryKey: queryKeys.payables,
    queryFn: () => getTransactions(),
  })
  const {
    data: financialSummary,
    isLoading: isSummaryLoading,
  } = useQuery({
    queryKey: queryKeys.summary,
    queryFn: () => getFinancialSummary(monthRange.startDate, monthRange.endDate),
  })
  const {
    data: creditCards = [],
    isLoading: isCardsLoading,
  } = useQuery({
    queryKey: queryKeys.creditCards,
    queryFn: getCreditCards,
  })
  const {
    data: financialAccounts = [],
    isLoading: isAccountsLoading,
  } = useQuery({
    queryKey: queryKeys.accounts,
    queryFn: getFinancialAccounts,
  })
  const createTransactionMutation = useMutation({
    mutationFn: createTransaction,
    onError: (error, variables) => {
      if (variables.status === "PAID") {
        setTransactionDialogError(getFriendlyErrorMessage(error))
      }
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["financial", "transactions"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts-payable"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "summary"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "credit-cards"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts"] }),
        queryClient.invalidateQueries({ queryKey: ["notifications"] }),
      ])
      setDialogOpen(false)
      setPaymentDialogOpen(false)
      setTransactionToEdit(null)
      setAccountPayableToEdit(null)
      setTransactionDialogError("")
    },
  })
  const updateTransactionMutation = useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string
      data: CreateTransactionInput
    }) => updateTransaction(id, data),
    onError: (error) => {
      setTransactionDialogError(getFriendlyErrorMessage(error))
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["financial", "transactions"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts-payable"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "summary"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "credit-cards"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts"] }),
        queryClient.invalidateQueries({ queryKey: ["notifications"] }),
      ])
      setDialogOpen(false)
      setTransactionToEdit(null)
      setTransactionDialogError("")
    },
  })
  const updateAccountPayableMutation = useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string
      data: CreateTransactionInput
    }) => updateAccountPayable(id, data),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["financial", "transactions"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts-payable"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "summary"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "credit-cards"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts"] }),
        queryClient.invalidateQueries({ queryKey: ["notifications"] }),
      ])
      setPaymentDialogOpen(false)
      setAccountPayableToEdit(null)
    },
  })
  const payAccountPayableMutation = useMutation({
    mutationFn: ({
      id,
      dueDate,
    }: {
      id: string
      dueDate: string
    }) => payAccountPayable(id, dueDate),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["financial", "transactions"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts-payable"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "summary"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "credit-cards"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts"] }),
        queryClient.invalidateQueries({ queryKey: ["notifications"] }),
      ])
    },
  })
  const payCreditCardDebtMutation = useMutation({
    mutationFn: ({
      cardId,
      data,
    }: {
      cardId: string
      data: PayCreditCardDebtInput
    }) => payCreditCardDebt(cardId, data),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["financial", "transactions"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts-payable"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "summary"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "credit-cards"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts"] }),
        queryClient.invalidateQueries({ queryKey: ["notifications"] }),
      ])
      setCreditCardToPay(null)
    },
  })
  const deleteTransactionMutation = useMutation({
    mutationFn: deleteTransaction,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["financial", "transactions"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts-payable"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "summary"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "credit-cards"] }),
        queryClient.invalidateQueries({ queryKey: ["financial", "accounts"] }),
        queryClient.invalidateQueries({ queryKey: ["notifications"] }),
      ])
      setTransactionToDelete(null)
    },
  })
  const createCreditCardMutation = useMutation<
    FinancialAccount | CreditCardSummary,
    Error,
    CardAccountFormInput
  >({
    mutationFn: (data: CardAccountFormInput) => {
      if (data.type === "CREDIT_CARD") {
        return createCreditCard({
          name: data.name,
          bank: data.institution,
          last4: data.last4 ?? "",
          cutoffDay: data.cutoffDay,
          paymentDay: data.paymentDay,
          limitCents: data.creditLimitCents,
          color: data.color,
        })
      }
      return createFinancialAccount(data)
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.creditCards })
      await queryClient.invalidateQueries({ queryKey: queryKeys.accounts })
      await queryClient.invalidateQueries({ queryKey: ["notifications"] })
      setAddCardDialogOpen(false)
      setCreditCardToEdit(null)
      setDebitCardToEdit(null)
    },
  })
  const updateCreditCardMutation = useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string
      data: CardAccountFormInput
    }) => {
      if (data.type !== "CREDIT_CARD") {
        throw new Error("invalid credit card data")
      }
      return updateCreditCard(id, {
        name: data.name,
        bank: data.institution,
        last4: data.last4 ?? "",
        cutoffDay: data.cutoffDay,
        paymentDay: data.paymentDay,
        limitCents: data.creditLimitCents,
        color: data.color,
      })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.creditCards })
      await queryClient.invalidateQueries({ queryKey: queryKeys.accounts })
      await queryClient.invalidateQueries({ queryKey: ["notifications"] })
      setAddCardDialogOpen(false)
      setCreditCardToEdit(null)
    },
  })
  const updateFinancialAccountMutation = useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string
      data: CreateFinancialAccountInput
    }) => updateFinancialAccount(id, data),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.accounts })
      await queryClient.invalidateQueries({ queryKey: queryKeys.creditCards })
      await queryClient.invalidateQueries({ queryKey: ["notifications"] })
      setAddCardDialogOpen(false)
      setDebitCardToEdit(null)
      setCreditCardToEdit(null)
    },
  })
  const deleteCreditCardMutation = useMutation({
    mutationFn: deleteCreditCard,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.creditCards })
      await queryClient.invalidateQueries({ queryKey: queryKeys.accounts })
      setCreditCardToDelete(null)
    },
  })
  const deleteFinancialAccountMutation = useMutation({
    mutationFn: deleteFinancialAccount,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.accounts })
      await queryClient.invalidateQueries({ queryKey: queryKeys.creditCards })
      setDebitCardToDelete(null)
    },
  })

  const transactions = useMemo(
    () =>
      apiTransactions
        .filter(
          (transaction) =>
            transaction.status !== "PENDING" &&
            !(
              transaction.status === "COMPLETED" &&
              transaction.recurrenceLimit === 0
            ),
        )
        .map((transaction) => mapTransaction(transaction, financialAccounts)),
    [apiTransactions, financialAccounts],
  )
  const manualPendingPayments = useMemo(
    () =>
      apiPayables
        .filter(
          (transaction) =>
            transaction.type === "EXPENSE" && transaction.status === "PENDING",
        )
        .flatMap(mapPendingPayments),
    [apiPayables],
  )
  const creditCardPendingPayments = useMemo(
    () =>
      creditCards
        .map(mapCreditCardPayable)
        .filter((payment): payment is PendingPayment => Boolean(payment)),
    [creditCards],
  )
  const pendingPayments = useMemo(
    () => [...manualPendingPayments, ...creditCardPendingPayments],
    [creditCardPendingPayments, manualPendingPayments],
  )
  const cashAccount = useMemo(
    () => financialAccounts.find((account) => account.type === "CASH") ?? null,
    [financialAccounts],
  )
  const debitCards = useMemo(
    () => financialAccounts.filter((account) => account.type === "DEBIT_CARD"),
    [financialAccounts],
  )
  const paymentSourceAccounts = useMemo(
    () =>
      financialAccounts.filter(
        (account) => account.type === "CASH" || account.type === "DEBIT_CARD",
      ),
    [financialAccounts],
  )
  const totalIncome = centsToAmount(financialSummary?.totalIncomeCents ?? 0)
  const totalExpense = centsToAmount(financialSummary?.totalExpenseCents ?? 0)
  const profitMargin = centsToAmount(financialSummary?.profitMarginCents ?? 0)
  const availableIncomePercentage =
    totalIncome > 0 ? ((profitMargin / totalIncome) * 100).toFixed(1) : "0.0"
  const transactionToDeleteLabel = transactionToDelete?.concept ?? "este movimiento"

  function handleDeleteTransaction(transactionId: string) {
    const transaction =
      apiTransactions.find(
        (currentTransaction) => currentTransaction.id === transactionId,
      ) ??
      apiPayables.find(
        (currentTransaction) => currentTransaction.id === transactionId,
      )
    if (transaction) {
      setTransactionToDelete(transaction)
    }
  }

  function handleEditTransaction(transactionId: string) {
    const transaction = apiTransactions.find(
      (currentTransaction) => currentTransaction.id === transactionId,
    )
    if (transaction) {
      setTransactionDialogError("")
      setTransactionToEdit(transaction)
      setDialogOpen(true)
    }
  }

  function handleEditAccountPayable(transactionId: string) {
    const transaction = apiPayables.find(
      (currentTransaction) => currentTransaction.id === transactionId,
    )
    if (transaction) {
      setAccountPayableToEdit(transaction)
      setPaymentDialogOpen(true)
    }
  }

  function handleSubmitTransaction(data: CreateTransactionInput) {
    setTransactionDialogError("")
    if (transactionToEdit?.id) {
      updateTransactionMutation.mutate({ id: transactionToEdit.id, data })
      return
    }

    createTransactionMutation.mutate(data)
  }

  function handleSubmitAccountPayable(data: CreateTransactionInput) {
    if (accountPayableToEdit?.id) {
      updateAccountPayableMutation.mutate({
        id: accountPayableToEdit.id,
        data: {
          ...data,
          type: "EXPENSE",
          status: "PENDING",
        },
      })
      return
    }

    createTransactionMutation.mutate(data)
  }

  function handleSubmitCard(data: CardAccountFormInput) {
    if (creditCardToEdit?.id) {
      updateCreditCardMutation.mutate({ id: creditCardToEdit.id, data })
      return
    }
    if (debitCardToEdit?.id) {
      updateFinancialAccountMutation.mutate({ id: debitCardToEdit.id, data })
      return
    }

    createCreditCardMutation.mutate(data)
  }

  function handlePayAccountPayable(transactionId: string, dueDate: string) {
    payAccountPayableMutation.mutate({ id: transactionId, dueDate })
  }

  function handleOpenCreditCardPayment(creditCardId?: string) {
    const card = creditCards.find((currentCard) => currentCard.id === creditCardId)
    if (card) {
      setCreditCardToPay(card)
    }
  }

  function handleSubmitCreditCardPayment(
    cardId: string,
    data: PayCreditCardDebtInput,
  ) {
    payCreditCardDebtMutation.mutate({ cardId, data })
  }

  function handleCloseTransactionDialog() {
    setDialogOpen(false)
    setTransactionToEdit(null)
    setTransactionDialogError("")
  }

  function handleClosePaymentDialog() {
    setPaymentDialogOpen(false)
    setAccountPayableToEdit(null)
  }

  function handleCloseCardDialog() {
    setAddCardDialogOpen(false)
    setCreditCardToEdit(null)
    setDebitCardToEdit(null)
  }

  function handleConfirmDeleteTransaction() {
    if (!transactionToDelete) {
      return
    }

    deleteTransactionMutation.mutate(transactionToDelete.id)
  }

  function handleConfirmDeleteCreditCard() {
    if (!creditCardToDelete) {
      return
    }

    deleteCreditCardMutation.mutate(creditCardToDelete.id)
  }

  function handleConfirmDeleteDebitCard() {
    if (!debitCardToDelete) {
      return
    }

    deleteFinancialAccountMutation.mutate(debitCardToDelete.id)
  }

  return (
    <main className="pwa-safe-bottom flex h-full min-h-0 flex-col gap-6 overflow-y-auto bg-slate-50 p-4 dark:bg-background sm:p-6 md:gap-8 md:p-8">
      <header className="flex flex-col gap-1">
        <h1 className="text-balance text-3xl font-bold tracking-tight text-foreground">
          Control financiero
        </h1>
        <p className="text-sm text-muted-foreground">
          Radiografia mensual de tus ingresos, egresos y liquidez.
        </p>
      </header>

      <div className="grid grid-cols-1 gap-8 xl:grid-cols-3">
        <div className="xl:col-span-2">
          <Card className="overflow-hidden border border-border/40 shadow-sm ring-0">
            <CardHeader className="p-6 pb-4">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="flex flex-col gap-1">
                  <div className="flex items-center gap-2">
                    <Wallet className="size-5 text-muted-foreground" />
                    <CardTitle className="text-base font-semibold">
                      Libro de Registro
                    </CardTitle>
                  </div>
                  <CardDescription>
                    {monthRange.label} - todas las cuentas
                  </CardDescription>
                </div>
                <Button
                  size="sm"
                  onClick={() => {
                    setTransactionDialogError("")
                    setTransactionToEdit(null)
                    setDialogOpen(true)
                  }}
                  className="shrink-0 h-9 rounded-lg text-sm font-semibold"
                >
                  <Plus data-icon="inline-start" />
                  <span className="hidden sm:inline">Nuevo Movimiento</span>
                  <span className="sm:hidden">Nuevo</span>
                </Button>
              </div>
            </CardHeader>
            <CardContent className="p-6 pt-0">
              <Tabs defaultValue="history" className="flex flex-col gap-0">
                <TabsList
                  variant="line"
                  className="h-auto w-full justify-start gap-0 rounded-none border-b border-border/40 bg-transparent p-0"
                >
                  <TabsTrigger
                    value="history"
                    className="h-auto flex-none rounded-none border-x-0 border-t-0 border-b-2 border-transparent bg-transparent px-4 py-2 text-sm font-medium text-muted-foreground shadow-none transition-colors hover:text-foreground focus-visible:ring-0 focus-visible:ring-offset-0 data-[state=active]:border-x-0 data-[state=active]:border-t-0 data-[state=active]:border-b-primary data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none"
                  >
                    Historial
                  </TabsTrigger>
                  <TabsTrigger
                    value="cards"
                    className="h-auto flex-none rounded-none border-x-0 border-t-0 border-b-2 border-transparent bg-transparent px-4 py-2 text-sm font-medium text-muted-foreground shadow-none transition-colors hover:text-foreground focus-visible:ring-0 focus-visible:ring-offset-0 data-[state=active]:border-x-0 data-[state=active]:border-t-0 data-[state=active]:border-b-primary data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none"
                  >
                    Billetera
                  </TabsTrigger>
                </TabsList>

                <TabsContent
                  value="history"
                  className="mt-0 pt-4 data-[state=inactive]:hidden"
                >
                  <Table>
                    <TableHeader>
                      <TableRow className="hover:bg-transparent">
                        <TableHead className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                          Fecha
                        </TableHead>
                        <TableHead className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                          Concepto
                        </TableHead>
                        <TableHead className="hidden text-xs font-medium uppercase tracking-wide text-muted-foreground md:table-cell">
                          Categoria
                        </TableHead>
                        <TableHead className="hidden text-xs font-medium uppercase tracking-wide text-muted-foreground lg:table-cell">
                          Metodo
                        </TableHead>
                        <TableHead className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                          Tipo
                        </TableHead>
                        <TableHead className="text-right text-xs font-medium uppercase tracking-wide text-muted-foreground">
                          Monto
                        </TableHead>
                        <TableHead className="w-10 text-right text-xs font-medium uppercase tracking-wide text-muted-foreground">
                          Acciones
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {isTransactionsLoading
                        ? Array.from({ length: 5 }).map((_, index) => (
                            <TableRow key={index}>
                              <TableCell>
                                <Skeleton className="h-4 w-full" />
                              </TableCell>
                              <TableCell>
                                <Skeleton className="h-4 w-full" />
                              </TableCell>
                              <TableCell className="hidden md:table-cell">
                                <Skeleton className="h-4 w-full" />
                              </TableCell>
                              <TableCell className="hidden lg:table-cell">
                                <Skeleton className="h-4 w-full" />
                              </TableCell>
                              <TableCell>
                                <Skeleton className="h-4 w-full" />
                              </TableCell>
                              <TableCell>
                                <Skeleton className="ml-auto h-4 w-full" />
                              </TableCell>
                              <TableCell>
                                <Skeleton className="ml-auto h-8 w-8" />
                              </TableCell>
                            </TableRow>
                          ))
                        : transactions.map((transaction) => (
                            <TableRow key={transaction.id}>
                              <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                                {transaction.date}
                              </TableCell>
                              <TableCell className="font-medium">
                                <span>{transaction.concept}</span>
                                {transaction.msi ? (
                                  <Badge className="ml-2 border-0 bg-blue-100 text-xs text-blue-700 hover:bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400">
                                    {transaction.msi}
                                  </Badge>
                                ) : null}
                              </TableCell>
                              <TableCell className="hidden text-muted-foreground md:table-cell">
                                <Badge variant="secondary" className="font-normal">
                                  {transaction.category}
                                </Badge>
                              </TableCell>
                              <TableCell className="hidden text-muted-foreground lg:table-cell">
                                <Badge variant="outline" className="max-w-48 truncate font-normal">
                                  {transaction.paymentMethod}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                {transaction.type === "income" ? (
                                  <span className="flex items-center gap-1 text-sm font-medium text-emerald-600 dark:text-emerald-400">
                                    <ArrowUpRight className="size-4" />
                                    Ingreso
                                  </span>
                                ) : (
                                  <span className="flex items-center gap-1 text-sm text-muted-foreground">
                                    <ArrowDownRight className="size-4" />
                                    Egreso
                                  </span>
                                )}
                              </TableCell>
                              <TableCell
                                className={cn(
                                  "text-right font-semibold tabular-nums",
                                  transaction.type === "income"
                                    ? "text-emerald-600 dark:text-emerald-400"
                                    : "text-foreground",
                                )}
                              >
                                {transaction.type === "income" ? "+" : "-"}
                                {fmt(transaction.amount)}
                              </TableCell>
                              <TableCell className="text-right">
                                <div className="flex justify-end gap-1">
                                  <Button
                                    size="icon"
                                    variant="ghost"
                                    className="size-8 cursor-pointer text-muted-foreground hover:text-foreground"
                                    aria-label={`Editar movimiento ${transaction.concept}`}
                                    disabled={updateTransactionMutation.isPending}
                                    onClick={() => handleEditTransaction(transaction.id)}
                                  >
                                    <Pencil className="size-4" />
                                  </Button>
                                <Button
                                  size="icon"
                                  variant="ghost"
                                    className="size-8 cursor-pointer text-muted-foreground hover:text-red-600"
                                  aria-label={`Eliminar movimiento ${transaction.concept}`}
                                  disabled={deleteTransactionMutation.isPending}
                                  onClick={() => handleDeleteTransaction(transaction.id)}
                                >
                                  <Trash2 className="size-4" />
                                </Button>
                                </div>
                              </TableCell>
                            </TableRow>
                          ))}
                    </TableBody>
                  </Table>
                </TabsContent>

                <TabsContent
                  value="cards"
                  className="mt-0 pt-4 data-[state=inactive]:hidden"
                >
                  <div className="grid grid-cols-1 gap-4 pt-4 sm:gap-6 md:grid-cols-2">
                    {isCardsLoading || isAccountsLoading ? (
                      Array.from({ length: 3 }).map((_, index) => (
                        <Skeleton
                          key={index}
                          className="aspect-[1.586/1] w-full rounded-xl"
                        />
                      ))
                    ) : (
                      <>
                        {cashAccount ? (
                          <div
                            className="group relative flex aspect-[1.586/1] w-full flex-col justify-between overflow-hidden rounded-xl border border-emerald-500 bg-gradient-to-br from-emerald-600 to-slate-900 p-6 text-white shadow-lg"
                          >
                            <div
                              className="pointer-events-none absolute inset-0 z-10 bg-black/0 transition-colors duration-150 group-hover:bg-black/35"
                              aria-hidden="true"
                            />
                            <div className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center gap-3 opacity-0 transition-opacity duration-150 group-hover:opacity-100">
                              <Button
                                type="button"
                                variant="secondary"
                                size="icon"
                                className="pointer-events-auto size-10 bg-white/90 text-slate-900 shadow-lg hover:bg-white"
                                aria-label={`Ver detalle de efectivo ${cashAccount.name}`}
                                title="Ver detalle"
                                onClick={() => setAccountDetail(cashAccount)}
                              >
                                <Eye className="size-4" />
                              </Button>
                            </div>

                            <div className="flex items-start justify-between">
                              <div className="flex flex-col gap-0.5">
                                <Wallet className="size-8 opacity-90" />
                                <span className="mt-2 text-xs font-medium opacity-70">
                                  Dinero disponible
                                </span>
                              </div>
                              <span className="text-xs font-bold uppercase tracking-widest opacity-60">
                                EFECTIVO
                              </span>
                            </div>

                            <div className="flex flex-col gap-1">
                              <span className="text-[10px] uppercase tracking-wider opacity-50">
                                Cuenta
                              </span>
                              <span className="text-lg font-semibold">
                                {cashAccount.name}
                              </span>
                            </div>

                            <div className="flex items-end justify-between">
                              <div className="flex flex-col gap-0.5">
                                <span className="text-[10px] uppercase tracking-wider opacity-50">
                                  Tipo
                                </span>
                                <span className="text-sm font-semibold">
                                  Cartera fisica
                                </span>
                              </div>
                              <div className="flex flex-col items-end gap-0.5">
                                <span className="text-[10px] uppercase tracking-wider opacity-50">
                                  Saldo actual
                                </span>
                                <span className="text-sm font-bold">
                                  {fmt(centsToAmount(cashAccount.currentBalanceCents))}
                                </span>
                              </div>
                            </div>
                          </div>
                        ) : null}

                        {creditCards.map((card) => {
                          const visual = cardVisualForColor(card.color)
                          const detailAccount = financialAccountFromCreditCard(
                            card,
                            financialAccounts,
                          )

                          return (
                            <div
                              key={card.id}
                              className={cn(
                                "group relative flex aspect-[1.586/1] w-full flex-col justify-between overflow-hidden rounded-xl border bg-gradient-to-br p-6 text-white shadow-lg",
                                visual.gradient,
                                visual.border,
                              )}
                            >
                              <div
                                className="pointer-events-none absolute inset-0 z-10 bg-black/0 transition-colors duration-150 group-hover:bg-black/35"
                                aria-hidden="true"
                              />
                              <div className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center gap-3 opacity-0 transition-opacity duration-150 group-hover:opacity-100">
                                <Button
                                  type="button"
                                  variant="secondary"
                                  size="icon"
                                  className="pointer-events-auto size-10 bg-white/90 text-slate-900 shadow-lg hover:bg-white"
                                  aria-label={`Ver detalle de tarjeta ${card.name}`}
                                  title="Ver detalle"
                                  onClick={() => setAccountDetail(detailAccount)}
                                >
                                  <Eye className="size-4" />
                                </Button>
                                <Button
                                  type="button"
                                  variant="secondary"
                                  size="icon"
                                  className="pointer-events-auto size-10 bg-white/90 text-slate-900 shadow-lg hover:bg-white"
                                  aria-label={`Editar tarjeta ${card.name}`}
                                  title="Editar tarjeta"
                                  disabled={updateCreditCardMutation.isPending}
                                  onClick={() => {
                                    setCreditCardToEdit(card)
                                    setAddCardDialogOpen(true)
                                  }}
                                >
                                  <Pencil className="size-4" />
                                </Button>
                                <Button
                                  type="button"
                                  variant="secondary"
                                  size="icon"
                                  className="pointer-events-auto size-10 bg-white/90 text-red-600 shadow-lg hover:bg-white hover:text-red-700"
                                  aria-label={`Eliminar tarjeta ${card.name}`}
                                  title="Eliminar tarjeta"
                                  disabled={deleteCreditCardMutation.isPending}
                                  onClick={() => setCreditCardToDelete(card)}
                                >
                                  <Trash2 className="size-4" />
                                </Button>
                              </div>
                              <div className="flex items-start justify-between">
                                <div className="flex flex-col gap-0.5">
                                  <CreditCard className="size-8 opacity-90" />
                                  <span className="mt-2 text-xs font-medium opacity-70">
                                    {card.bank}
                                  </span>
                                </div>
                                <span className="text-xs font-bold uppercase tracking-widest opacity-60">
                                  VISA
                                </span>
                              </div>

                              <div className="font-mono text-lg font-semibold tracking-[0.3em] opacity-90">
                                .... .... .... {card.last4}
                              </div>

                              <div className="flex items-end justify-between">
                                <div className="flex flex-col gap-0.5">
                                  <span className="text-[10px] uppercase tracking-wider opacity-50">
                                    Fecha de corte
                                  </span>
                                  <span className="flex items-center gap-1 text-sm font-semibold">
                                    <Calendar className="size-3.5 opacity-70" />
                                    {cardCutoffLabel(card.cutoffDay)}
                                  </span>
                                  <span className="text-[10px] uppercase tracking-wider opacity-50">
                                    Pago dia {card.paymentDay}
                                  </span>
                                </div>
                                <div className="flex flex-col items-end gap-0.5">
                                  <span className="text-[10px] uppercase tracking-wider opacity-50">
                                    Deuda actual
                                  </span>
                                  <span className="text-sm font-bold">
                                    {fmt(centsToAmount(card.currentDebtCents))}
                                  </span>
                                  <span className="text-[10px] uppercase tracking-wider opacity-50">
                                    Limite {fmt(centsToAmount(card.limitCents))}
                                  </span>
                                </div>
                              </div>
                            </div>
                          )
                        })}

                        {debitCards.map((card) => {
                          const visual = cardVisualForColor(card.color)

                          return (
                            <div
                              key={card.id}
                              className={cn(
                                "group relative flex aspect-[1.586/1] w-full flex-col justify-between overflow-hidden rounded-xl border bg-gradient-to-br p-6 text-white shadow-lg",
                                visual.gradient,
                                visual.border,
                              )}
                            >
                              <div
                                className="pointer-events-none absolute inset-0 z-10 bg-black/0 transition-colors duration-150 group-hover:bg-black/35"
                                aria-hidden="true"
                              />
                              <div className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center gap-3 opacity-0 transition-opacity duration-150 group-hover:opacity-100">
                                <Button
                                  type="button"
                                  variant="secondary"
                                  size="icon"
                                  className="pointer-events-auto size-10 bg-white/90 text-slate-900 shadow-lg hover:bg-white"
                                  aria-label={`Ver detalle de tarjeta ${card.name}`}
                                  title="Ver detalle"
                                  onClick={() => setAccountDetail(card)}
                                >
                                  <Eye className="size-4" />
                                </Button>
                                <Button
                                  type="button"
                                  variant="secondary"
                                  size="icon"
                                  className="pointer-events-auto size-10 bg-white/90 text-slate-900 shadow-lg hover:bg-white"
                                  aria-label={`Editar tarjeta ${card.name}`}
                                  title="Editar tarjeta"
                                  disabled={updateFinancialAccountMutation.isPending}
                                  onClick={() => {
                                    setDebitCardToEdit(card)
                                    setCreditCardToEdit(null)
                                    setAddCardDialogOpen(true)
                                  }}
                                >
                                  <Pencil className="size-4" />
                                </Button>
                                <Button
                                  type="button"
                                  variant="secondary"
                                  size="icon"
                                  className="pointer-events-auto size-10 bg-white/90 text-red-600 shadow-lg hover:bg-white hover:text-red-700"
                                  aria-label={`Eliminar tarjeta ${card.name}`}
                                  title="Eliminar tarjeta"
                                  disabled={deleteFinancialAccountMutation.isPending}
                                  onClick={() => setDebitCardToDelete(card)}
                                >
                                  <Trash2 className="size-4" />
                                </Button>
                              </div>
                              <div className="flex items-start justify-between">
                                <div className="flex flex-col gap-0.5">
                                  <CreditCard className="size-8 opacity-90" />
                                  <span className="mt-2 text-xs font-medium opacity-70">
                                    {card.institution || "Debito"}
                                  </span>
                                </div>
                                <span className="text-xs font-bold uppercase tracking-widest opacity-60">
                                  DEBITO
                                </span>
                              </div>

                              <div className="font-mono text-lg font-semibold tracking-[0.3em] opacity-90">
                                .... .... .... {card.last4 ?? "0000"}
                              </div>

                              <div className="flex items-end justify-between">
                                <div className="flex flex-col gap-0.5">
                                  <span className="text-[10px] uppercase tracking-wider opacity-50">
                                    Nombre
                                  </span>
                                  <span className="text-sm font-semibold">
                                    {card.name}
                                  </span>
                                  <span className="text-[10px] uppercase tracking-wider opacity-50">
                                    Saldo inicial {fmt(centsToAmount(card.openingBalanceCents))}
                                  </span>
                                </div>
                                <div className="flex flex-col items-end gap-0.5">
                                  <span className="text-[10px] uppercase tracking-wider opacity-50">
                                    Saldo actual
                                  </span>
                                  <span className="text-sm font-bold">
                                    {fmt(centsToAmount(card.currentBalanceCents))}
                                  </span>
                                </div>
                              </div>
                            </div>
                          )
                        })}

                        <button
                          type="button"
                          onClick={() => {
                            setCreditCardToEdit(null)
                            setDebitCardToEdit(null)
                            setAddCardDialogOpen(true)
                          }}
                          className="flex aspect-[1.586/1] w-full cursor-pointer flex-col items-center justify-center gap-3 rounded-xl border-2 border-dashed border-border bg-transparent text-muted-foreground transition-colors hover:border-foreground/30 hover:bg-muted/30 hover:text-foreground"
                        >
                          <PlusCircle className="size-7 opacity-50" />
                          <span className="text-sm font-medium">Agregar Tarjeta</span>
                        </button>
                      </>
                    )}
                  </div>
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>
        </div>

        <div className="flex flex-col gap-6 xl:col-span-1">
          <Card className="overflow-hidden border border-border/40 shadow-sm ring-0">
            <CardHeader className="p-6 pb-4">
              <div className="flex items-start justify-between gap-2">
                <div className="flex flex-col gap-1">
                  <div className="flex items-center gap-2">
                    <AlertCircle className="size-5 text-muted-foreground" />
                    <CardTitle className="text-base font-semibold">
                      Cuentas por Pagar
                    </CardTitle>
                  </div>
                  <CardDescription>
                    {isPayablesLoading ? (
                      <Skeleton className="h-4 w-32" />
                    ) : (
                      `${pendingPayments.length} cuentas por pagar`
                    )}
                  </CardDescription>
                </div>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="shrink-0 rounded-lg text-muted-foreground hover:text-foreground"
                  aria-label="Agregar cuenta por pagar"
                  onClick={() => {
                    setAccountPayableToEdit(null)
                    setPaymentDialogOpen(true)
                  }}
                >
                  <Plus />
                </Button>
              </div>
            </CardHeader>
            <CardContent className="p-6 pt-0">
              <div className="flex flex-col gap-4">
                {isPayablesLoading
                  ? Array.from({ length: 3 }).map((_, index) => (
                      <Skeleton
                        key={index}
                        className="h-16 w-full rounded-lg"
                      />
                    ))
                  : pendingPayments.map((payment) => {
                      const Icon = payment.icon
                      return (
                        <div
                          key={payment.id}
                          className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
                        >
                          <div className="flex min-w-0 items-center gap-3">
                            <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted">
                              <Icon className="size-4 text-muted-foreground" />
                            </div>
                            <div className="flex min-w-0 flex-col">
                              <span
                                className={cn(
                                  "truncate text-sm font-medium",
                                  payment.visualState === "overdue"
                                    ? "text-red-600 dark:text-red-500"
                                    : payment.visualState === "paid"
                                    ? "text-green-600 dark:text-green-500"
                                    : "text-foreground",
                                )}
                              >
                                {payment.service}
                              </span>
                              <span
                                className={cn(
                                  "text-xs",
                                  payment.visualState === "overdue"
                                    ? "font-medium text-red-600 dark:text-red-500"
                                    : payment.visualState === "paid"
                                    ? "font-medium text-green-600 dark:text-green-500"
                                    : "text-muted-foreground",
                                )}
                              >
                                Vence {payment.dueDate}
                              </span>
                            </div>
                          </div>
                          <div className="flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto sm:shrink-0">
                            <span
                              className={cn(
                                "text-sm font-semibold tabular-nums",
                                payment.visualState === "overdue"
                                  ? "text-red-600 dark:text-red-500"
                                  : payment.visualState === "paid"
                                  ? "text-green-600 dark:text-green-500"
                                  : "text-foreground",
                              )}
                            >
                              {fmt(payment.amount)}
                            </span>
                            <Button
                              size="sm"
                              variant="outline"
                              className="rounded-md"
                              disabled={
                                (payment.type === "manual"
                                  ? payAccountPayableMutation.isPending
                                  : payCreditCardDebtMutation.isPending) ||
                                payment.visualState === "paid"
                              }
                              onClick={() => {
                                if (payment.type === "credit_card") {
                                  handleOpenCreditCardPayment(payment.creditCardId)
                                  return
                                }
                                handlePayAccountPayable(
                                  payment.transactionId,
                                  payment.dueDateRaw,
                                )
                              }}
                            >
                              {payment.visualState === "paid" ? "Pagado" : "Pagar"}
                            </Button>
                            {payment.type === "manual" ? (
                              <>
                                <Button
                                  size="icon"
                                  variant="ghost"
                                  className="size-8 cursor-pointer text-muted-foreground hover:text-foreground"
                                  aria-label={`Editar cuenta por pagar ${payment.service}`}
                                  disabled={updateAccountPayableMutation.isPending}
                                  onClick={() =>
                                    handleEditAccountPayable(payment.transactionId)
                                  }
                                >
                                  <Pencil className="size-4" />
                                </Button>
                                <Button
                                  size="icon"
                                  variant="ghost"
                                  className="size-8 cursor-pointer text-muted-foreground hover:text-red-600"
                                  aria-label={`Eliminar cuenta por pagar ${payment.service}`}
                                  disabled={deleteTransactionMutation.isPending}
                                  onClick={() =>
                                    handleDeleteTransaction(payment.transactionId)
                                  }
                                >
                                  <Trash2 className="size-4" />
                                </Button>
                              </>
                            ) : null}
                          </div>
                        </div>
                      )
                    })}
              </div>
            </CardContent>
          </Card>

          <Card className="overflow-hidden border border-primary/20 bg-gradient-to-br from-primary/5 to-transparent shadow-sm ring-0">
            <CardHeader className="p-6 pb-4">
              <div className="flex items-center gap-2">
                <Wallet className="size-5 text-primary/70" />
                <CardTitle className="text-base font-semibold">
                  Resumen Mensual
                </CardTitle>
              </div>
              <CardDescription>{monthRange.label}</CardDescription>
            </CardHeader>
            <CardContent className="p-6 pt-0">
              <div className="flex flex-col gap-1">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">
                    Ingreso Total
                  </span>
                  {isSummaryLoading ? (
                    <Skeleton className="h-4 w-24" />
                  ) : (
                    <span className="text-sm font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">
                      +{fmt(totalIncome)}
                    </span>
                  )}
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">
                    Gasto Acumulado
                  </span>
                  {isSummaryLoading ? (
                    <Skeleton className="h-4 w-24" />
                  ) : (
                    <span className="text-sm font-semibold tabular-nums text-foreground">
                      -{fmt(totalExpense)}
                    </span>
                  )}
                </div>
              </div>

              <div className="my-4 h-px w-full bg-border" />

              <div className="flex flex-col gap-1">
                <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                  Margen de utilidad
                </span>
                {isSummaryLoading ? (
                  <Skeleton className="h-10 w-40" />
                ) : (
                  <span
                    className={cn(
                      "text-4xl font-bold tabular-nums",
                      profitMargin >= 0
                        ? "text-foreground"
                        : "text-red-500 dark:text-red-400",
                    )}
                  >
                    {fmt(profitMargin)}
                  </span>
                )}
                {isSummaryLoading ? (
                  <Skeleton className="h-4 w-48" />
                ) : (
                  <span className="text-xs text-muted-foreground">
                    {profitMargin >= 0
                      ? `${availableIncomePercentage}% de tus ingresos disponibles`
                      : "Deficit este mes"}
                  </span>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <NewMovementDialog
        open={dialogOpen}
        onClose={handleCloseTransactionDialog}
        onSubmit={handleSubmitTransaction}
        isSaving={createTransactionMutation.isPending || updateTransactionMutation.isPending}
        transaction={transactionToEdit}
        paymentAccounts={financialAccounts}
        submitError={transactionDialogError}
      />
      <NewPaymentDialog
        open={paymentDialogOpen}
        onClose={handleClosePaymentDialog}
        onSubmit={handleSubmitAccountPayable}
        isSaving={
          createTransactionMutation.isPending ||
          updateAccountPayableMutation.isPending
        }
        accountPayable={accountPayableToEdit}
      />
      <AddCardDialog
        open={addCardDialogOpen}
        onClose={handleCloseCardDialog}
        onSubmit={handleSubmitCard}
        isSaving={
          createCreditCardMutation.isPending ||
          updateCreditCardMutation.isPending ||
          updateFinancialAccountMutation.isPending
        }
        card={creditCardToEdit}
        debitCard={debitCardToEdit}
      />
      <AccountDetailModal
        open={Boolean(accountDetail)}
        account={accountDetail}
        onClose={() => setAccountDetail(null)}
      />
      <PayCreditCardDialog
        open={Boolean(creditCardToPay)}
        card={creditCardToPay}
        sourceAccounts={paymentSourceAccounts}
        onClose={() => setCreditCardToPay(null)}
        onSubmit={handleSubmitCreditCardPayment}
        isSaving={payCreditCardDebtMutation.isPending}
      />
      <ConfirmDialog
        open={Boolean(transactionToDelete)}
        onOpenChange={(open) => {
          if (!open) {
            setTransactionToDelete(null)
          }
        }}
        title="Eliminar movimiento"
        description={`Se eliminara "${transactionToDeleteLabel}" del control financiero. Esta accion no se puede deshacer.`}
        confirmLabel="Eliminar movimiento"
        isPending={deleteTransactionMutation.isPending}
        onConfirm={handleConfirmDeleteTransaction}
      />
      <ConfirmDialog
        open={Boolean(creditCardToDelete)}
        onOpenChange={(open) => {
          if (!open) {
            setCreditCardToDelete(null)
          }
        }}
        title="Eliminar tarjeta"
        description={
          creditCardToDelete
            ? `Se eliminara la tarjeta "${creditCardToDelete.name}" terminacion ${creditCardToDelete.last4}. Esta accion no se puede deshacer.`
            : ""
        }
        confirmLabel="Eliminar tarjeta"
        isPending={deleteCreditCardMutation.isPending}
        onConfirm={handleConfirmDeleteCreditCard}
      />
      <ConfirmDialog
        open={Boolean(debitCardToDelete)}
        onOpenChange={(open) => {
          if (!open) {
            setDebitCardToDelete(null)
          }
        }}
        title="Eliminar tarjeta"
        description={
          debitCardToDelete
            ? `Se eliminara la tarjeta "${debitCardToDelete.name}" terminacion ${debitCardToDelete.last4 ?? "0000"}. Esta accion no se puede deshacer.`
            : ""
        }
        confirmLabel="Eliminar tarjeta"
        isPending={deleteFinancialAccountMutation.isPending}
        onConfirm={handleConfirmDeleteDebitCard}
      />

      <div className="h-24" />
    </main>
  )
}
