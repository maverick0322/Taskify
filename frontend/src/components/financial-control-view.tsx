"use client"

import { useMemo, useState, type ElementType } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  AlertCircle,
  ArrowDownRight,
  ArrowUpRight,
  Building2,
  Calendar,
  CreditCard,
  Phone,
  Plus,
  PlusCircle,
  Wallet,
  Wifi,
  Zap,
} from "lucide-react"

import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  createTransaction,
  getFinancialSummary,
  getTransactions,
  type CreateTransactionInput,
  type FinancialTransaction,
} from "@/services/financial_api"

type TransactionType = "income" | "expense"

interface Transaction {
  id: string
  date: string
  concept: string
  category: string
  type: TransactionType
  amount: number
  msi?: string
}

interface PendingPayment {
  id: string
  service: string
  icon: ElementType
  dueDate: string
  isUrgent: boolean
  amount: number
}

interface CreditCardItem {
  id: string
  bank: string
  lastFour: string
  cutDate: string
  limit: number
  gradient: string
  border: string
}

const CREDIT_CARDS: CreditCardItem[] = [
  {
    id: "1",
    bank: "BBVA Bancomer",
    lastFour: "4821",
    cutDate: "22 Jun 2026",
    limit: 80000,
    gradient: "from-blue-700 to-blue-900",
    border: "border-blue-600",
  },
  {
    id: "2",
    bank: "Citibanamex",
    lastFour: "3047",
    cutDate: "28 Jun 2026",
    limit: 50000,
    gradient: "from-slate-700 to-slate-900",
    border: "border-slate-600",
  },
  {
    id: "3",
    bank: "HSBC",
    lastFour: "9163",
    cutDate: "05 Jul 2026",
    limit: 30000,
    gradient: "from-red-800 to-red-950",
    border: "border-red-700",
  },
  {
    id: "4",
    bank: "Santander",
    lastFour: "6205",
    cutDate: "10 Jul 2026",
    limit: 45000,
    gradient: "from-rose-700 to-rose-900",
    border: "border-rose-600",
  },
]

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

function mapTransaction(transaction: FinancialTransaction): Transaction {
  return {
    id: transaction.id,
    date: formatDisplayDate(transaction.date),
    concept: transaction.concept,
    category: transaction.category,
    type: transaction.type === "INCOME" ? "income" : "expense",
    amount: centsToAmount(transaction.amountCents),
    msi: transaction.msi ? `${transaction.msi} MSI` : undefined,
  }
}

function mapPendingPayment(transaction: FinancialTransaction): PendingPayment {
  return {
    id: transaction.id,
    service: transaction.concept,
    icon: iconForCategory(transaction.category),
    dueDate: formatDisplayDate(transaction.date),
    isUrgent: isUrgentPayment(transaction.date),
    amount: centsToAmount(transaction.amountCents),
  }
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

function isUrgentPayment(rawDate: string) {
  const [year, month, day] = rawDate.split("-").map(Number)
  if (!year || !month || !day) {
    return false
  }

  const dueDate = new Date(year, month - 1, day)
  const currentDate = new Date()
  const differenceInMilliseconds = dueDate.getTime() - currentDate.getTime()
  const differenceInDays = Math.ceil(differenceInMilliseconds / 86400000)

  return differenceInDays <= 7
}

function financialQueryKeys(startDate: string, endDate: string) {
  return {
    transactions: ["financial", "transactions", startDate, endDate] as const,
    summary: ["financial", "summary", startDate, endDate] as const,
  }
}

function NewMovementDialog({
  open,
  onClose,
  onSubmit,
  isSaving,
}: {
  open: boolean
  onClose: () => void
  onSubmit: (data: CreateTransactionInput) => void
  isSaving: boolean
}) {
  const [tipo, setTipo] = useState("")
  const [categoria, setCategoria] = useState("")
  const [recurrencia, setRecurrencia] = useState("once")
  const [monto, setMonto] = useState("")
  const [concepto, setConcepto] = useState("")

  const handleSubmit = () => {
    onSubmit({
      type: tipo === "income" ? "INCOME" : "EXPENSE",
      concept: concepto,
      category: categoria,
      amountCents: amountToCents(monto),
      date: formatDateInput(new Date()),
      status: "PAID",
      msi: null,
    })
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Nuevo Movimiento</DialogTitle>
          <DialogDescription>
            Registra un ingreso o egreso en tu libro financiero.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-5 py-2">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="monto">Monto</Label>
            <div className="relative">
              <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm font-medium text-muted-foreground">
                $
              </span>
              <Input
                id="monto"
                type="number"
                placeholder="0.00"
                className="pl-7"
                value={monto}
                onChange={(event) => setMonto(event.target.value)}
              />
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="concepto">Concepto</Label>
            <Input
              id="concepto"
              placeholder="Ej. Sueldo mensual, Netflix..."
              value={concepto}
              onChange={(event) => setConcepto(event.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
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
          </div>

          <div className="flex flex-col gap-3">
            <Label>Recurrencia</Label>
            <RadioGroup
              value={recurrencia}
              onValueChange={setRecurrencia}
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
                  <RadioGroupItem value={option.value} id={option.value} />
                  <span className="text-sm font-medium text-foreground">
                    {option.label}
                  </span>
                </label>
              ))}
            </RadioGroup>
          </div>
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
}: {
  open: boolean
  onClose: () => void
  onSubmit: (data: CreateTransactionInput) => void
  isSaving: boolean
}) {
  const [categoria, setCategoria] = useState("")
  const [recurrencia, setRecurrencia] = useState("monthly")
  const [servicio, setServicio] = useState("")
  const [monto, setMonto] = useState("")
  const [fechaVence, setFechaVence] = useState("")

  const handleSubmit = () => {
    const selectedService = SERVICES_ICONS.find((service) => service.value === categoria)

    onSubmit({
      type: "EXPENSE",
      concept: servicio,
      category: selectedService?.label ?? categoria,
      amountCents: amountToCents(monto),
      date: fechaVence,
      status: "PENDING",
      msi: null,
    })
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Nueva Cuenta por Pagar</DialogTitle>
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

          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="monto-pago">Monto</Label>
              <div className="relative">
                <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm font-medium text-muted-foreground">
                  $
                </span>
                <Input
                  id="monto-pago"
                  type="number"
                  placeholder="0.00"
                  className="pl-7"
                  value={monto}
                  onChange={(event) => setMonto(event.target.value)}
                />
              </div>
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
              onValueChange={setRecurrencia}
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
}: {
  open: boolean
  onClose: () => void
}) {
  const [cardType, setCardType] = useState<"credit" | "debit">("credit")
  const [bank, setBank] = useState("")
  const [selectedGradient, setSelectedGradient] = useState(CARD_GRADIENTS[0])

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Agregar Tarjeta</DialogTitle>
          <DialogDescription>
            Vincula una tarjeta de credito o debito a tu control financiero.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-5 py-2">
          <div className="flex flex-col gap-2">
            <Label>Tipo de tarjeta</Label>
            <div className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={() => setCardType("credit")}
                className={cn(
                  "flex cursor-pointer flex-col items-center gap-2 rounded-xl border-2 px-4 py-4 transition-all",
                  cardType === "credit"
                    ? "border-primary bg-primary/5 text-foreground"
                    : "border-border bg-transparent text-muted-foreground hover:bg-muted/40",
                )}
              >
                <CreditCard className="size-6" />
                <span className="text-sm font-semibold">Credito</span>
                <span className="text-center text-xs leading-tight opacity-70">
                  Limite de credito y fecha de corte
                </span>
              </button>
              <button
                type="button"
                onClick={() => setCardType("debit")}
                className={cn(
                  "flex cursor-pointer flex-col items-center gap-2 rounded-xl border-2 px-4 py-4 transition-all",
                  cardType === "debit"
                    ? "border-primary bg-primary/5 text-foreground"
                    : "border-border bg-transparent text-muted-foreground hover:bg-muted/40",
                )}
              >
                <Wallet className="size-6" />
                <span className="text-sm font-semibold">Debito</span>
                <span className="text-center text-xs leading-tight opacity-70">
                  Saldo disponible en cuenta
                </span>
              </button>
            </div>
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

          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="last-four">Ultimos 4 digitos</Label>
              <Input
                id="last-four"
                placeholder="0000"
                maxLength={4}
                className="font-mono tracking-widest"
              />
            </div>
            {cardType === "credit" ? (
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="cut-date">Fecha de corte</Label>
                <Input id="cut-date" type="date" />
              </div>
            ) : (
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="balance">Saldo disponible</Label>
                <div className="relative">
                  <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm font-medium text-muted-foreground">
                    $
                  </span>
                  <Input id="balance" type="number" placeholder="0.00" className="pl-7" />
                </div>
              </div>
            )}
          </div>

          {cardType === "credit" ? (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="limit">Limite de credito</Label>
              <div className="relative">
                <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm font-medium text-muted-foreground">
                  $
                </span>
                <Input id="limit" type="number" placeholder="0.00" className="pl-7" />
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
        </div>

        <DialogFooter showCloseButton>
          <Button type="submit" className="w-full sm:w-auto">
            Agregar tarjeta
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
  const queryClient = useQueryClient()
  const monthRange = useMemo(() => currentMonthRange(), [])
  const queryKeys = useMemo(
    () => financialQueryKeys(monthRange.startDate, monthRange.endDate),
    [monthRange.endDate, monthRange.startDate],
  )

  const { data: apiTransactions = [] } = useQuery({
    queryKey: queryKeys.transactions,
    queryFn: () =>
      getTransactions({
        startDate: monthRange.startDate,
        endDate: monthRange.endDate,
      }),
  })
  const { data: financialSummary } = useQuery({
    queryKey: queryKeys.summary,
    queryFn: () => getFinancialSummary(monthRange.startDate, monthRange.endDate),
  })
  const createTransactionMutation = useMutation({
    mutationFn: createTransaction,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.transactions }),
        queryClient.invalidateQueries({ queryKey: queryKeys.summary }),
      ])
      setDialogOpen(false)
      setPaymentDialogOpen(false)
    },
  })

  const transactions = useMemo(
    () => apiTransactions.map(mapTransaction),
    [apiTransactions],
  )
  const pendingPayments = useMemo(
    () =>
      apiTransactions
        .filter(
          (transaction) =>
            transaction.type === "EXPENSE" && transaction.status === "PENDING",
        )
        .map(mapPendingPayment),
    [apiTransactions],
  )
  const totalIncome = centsToAmount(financialSummary?.totalIncomeCents ?? 0)
  const totalExpense = centsToAmount(financialSummary?.totalExpenseCents ?? 0)
  const profitMargin = centsToAmount(financialSummary?.profitMarginCents ?? 0)
  const availableIncomePercentage =
    totalIncome > 0 ? ((profitMargin / totalIncome) * 100).toFixed(1) : "0.0"

  return (
    <main className="flex h-full min-h-screen flex-col gap-8 overflow-y-auto bg-slate-50 p-8 dark:bg-background">
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
              <div className="flex items-start justify-between gap-4">
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
                  onClick={() => setDialogOpen(true)}
                  className="shrink-0"
                >
                  <Plus data-icon="inline-start" />
                  Nuevo Movimiento
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
                    className="h-auto flex-none rounded-none border-b-2 border-transparent bg-transparent px-4 py-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none"
                  >
                    Historial
                  </TabsTrigger>
                  <TabsTrigger
                    value="cards"
                    className="h-auto flex-none rounded-none border-b-2 border-transparent bg-transparent px-4 py-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none"
                  >
                    Tarjetas
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
                        <TableHead className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                          Tipo
                        </TableHead>
                        <TableHead className="text-right text-xs font-medium uppercase tracking-wide text-muted-foreground">
                          Monto
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {transactions.map((transaction) => (
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
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TabsContent>

                <TabsContent
                  value="cards"
                  className="mt-0 pt-4 data-[state=inactive]:hidden"
                >
                  <div className="grid grid-cols-1 gap-6 pt-4 md:grid-cols-2">
                    {CREDIT_CARDS.map((card) => (
                      <div
                        key={card.id}
                        className={cn(
                          "flex aspect-[1.586/1] w-full flex-col justify-between rounded-xl border bg-gradient-to-br p-6 text-white shadow-lg",
                          card.gradient,
                          card.border,
                        )}
                      >
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
                          .... .... .... {card.lastFour}
                        </div>

                        <div className="flex items-end justify-between">
                          <div className="flex flex-col gap-0.5">
                            <span className="text-[10px] uppercase tracking-wider opacity-50">
                              Fecha de corte
                            </span>
                            <span className="flex items-center gap-1 text-sm font-semibold">
                              <Calendar className="size-3.5 opacity-70" />
                              {card.cutDate}
                            </span>
                          </div>
                          <div className="flex flex-col items-end gap-0.5">
                            <span className="text-[10px] uppercase tracking-wider opacity-50">
                              Limite
                            </span>
                            <span className="text-sm font-bold">
                              {fmt(card.limit)}
                            </span>
                          </div>
                        </div>
                      </div>
                    ))}

                    <button
                      type="button"
                      onClick={() => setAddCardDialogOpen(true)}
                      className="flex aspect-[1.586/1] w-full cursor-pointer flex-col items-center justify-center gap-3 rounded-xl border-2 border-dashed border-border bg-transparent text-muted-foreground transition-colors hover:border-foreground/30 hover:bg-muted/30 hover:text-foreground"
                    >
                      <PlusCircle className="size-7 opacity-50" />
                      <span className="text-sm font-medium">Agregar Tarjeta</span>
                    </button>
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
                    {pendingPayments.length} pendientes este mes
                  </CardDescription>
                </div>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="shrink-0 text-muted-foreground hover:text-foreground"
                  aria-label="Agregar cuenta por pagar"
                  onClick={() => setPaymentDialogOpen(true)}
                >
                  <Plus />
                </Button>
              </div>
            </CardHeader>
            <CardContent className="p-6 pt-0">
              <div className="flex flex-col gap-4">
                {pendingPayments.map((payment) => {
                  const Icon = payment.icon
                  return (
                    <div
                      key={payment.id}
                      className="flex items-center justify-between gap-3"
                    >
                      <div className="flex min-w-0 items-center gap-3">
                        <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted">
                          <Icon className="size-4 text-muted-foreground" />
                        </div>
                        <div className="flex min-w-0 flex-col">
                          <span className="truncate text-sm font-medium text-foreground">
                            {payment.service}
                          </span>
                          <span
                            className={cn(
                              "text-xs",
                              payment.isUrgent
                                ? "font-medium text-red-500 dark:text-red-400"
                                : "text-muted-foreground",
                            )}
                          >
                            Vence {payment.dueDate}
                          </span>
                        </div>
                      </div>
                      <div className="flex shrink-0 items-center gap-2">
                        <span className="text-sm font-semibold tabular-nums text-foreground">
                          {fmt(payment.amount)}
                        </span>
                        <Button size="sm" variant="outline">
                          Pagar
                        </Button>
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
                  <span className="text-sm font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">
                    +{fmt(totalIncome)}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">
                    Gasto Acumulado
                  </span>
                  <span className="text-sm font-semibold tabular-nums text-foreground">
                    -{fmt(totalExpense)}
                  </span>
                </div>
              </div>

              <div className="my-4 h-px w-full bg-border" />

              <div className="flex flex-col gap-1">
                <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                  Margen de utilidad
                </span>
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
                <span className="text-xs text-muted-foreground">
                  {profitMargin >= 0
                    ? `${availableIncomePercentage}% de tus ingresos disponibles`
                    : "Deficit este mes"}
                </span>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <NewMovementDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        onSubmit={(data) => createTransactionMutation.mutate(data)}
        isSaving={createTransactionMutation.isPending}
      />
      <NewPaymentDialog
        open={paymentDialogOpen}
        onClose={() => setPaymentDialogOpen(false)}
        onSubmit={(data) => createTransactionMutation.mutate(data)}
        isSaving={createTransactionMutation.isPending}
      />
      <AddCardDialog
        open={addCardDialogOpen}
        onClose={() => setAddCardDialogOpen(false)}
      />

      <div className="h-24" />
    </main>
  )
}
