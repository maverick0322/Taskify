import * as React from "react"

import { cn } from "@/lib/utils"
import { Input } from "@/components/ui/input"

type CurrencyInputProps = Omit<
  React.ComponentProps<typeof Input>,
  "type" | "value" | "onChange"
> & {
  value: string
  onValueChange: (value: string) => void
}

function normalizeCurrencyValue(value: string) {
  const cleanedValue = value.replace(/[^\d.]/g, "")
  const [integerPart = "", ...decimalParts] = cleanedValue.split(".")
  const decimalPart = decimalParts.join("").slice(0, 2)
  const normalizedInteger = integerPart.replace(/^0+(?=\d)/, "")

  if (cleanedValue.includes(".")) {
    return `${normalizedInteger || "0"}.${decimalPart}`
  }

  return normalizedInteger
}

function formatCurrencyValue(value: string) {
  const normalizedValue = normalizeCurrencyValue(value)
  if (!normalizedValue) {
    return ""
  }

  const [integerPart = "0", decimalPart] = normalizedValue.split(".")
  const numericInteger = Number(integerPart || "0")
  const formattedInteger = Number.isFinite(numericInteger)
    ? new Intl.NumberFormat("en-US", {
        maximumFractionDigits: 0,
      }).format(numericInteger)
    : integerPart

  if (normalizedValue.includes(".")) {
    return `${formattedInteger}.${decimalPart ?? ""}`
  }

  return formattedInteger
}

function CurrencyInput({
  className,
  value,
  onValueChange,
  onBlur,
  onKeyDown,
  onPaste,
  placeholder = "0.00",
  ...props
}: CurrencyInputProps) {
  const displayValue = formatCurrencyValue(value)

  function updateValue(nextValue: string) {
    onValueChange(normalizeCurrencyValue(nextValue))
  }

  function handleKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    onKeyDown?.(event)
    if (event.defaultPrevented) {
      return
    }

    const allowedControlKeys = [
      "Backspace",
      "Delete",
      "Tab",
      "Enter",
      "Escape",
      "ArrowLeft",
      "ArrowRight",
      "ArrowUp",
      "ArrowDown",
      "Home",
      "End",
    ]

    if (event.ctrlKey || event.metaKey || allowedControlKeys.includes(event.key)) {
      return
    }

    if (/^\d$/.test(event.key) || event.key === ".") {
      return
    }

    event.preventDefault()
  }

  return (
    <div className="relative">
      <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm font-medium text-muted-foreground">
        $
      </span>
      <Input
        {...props}
        type="text"
        inputMode="decimal"
        placeholder={placeholder}
        className={cn("pl-7", className)}
        value={displayValue}
        onKeyDown={handleKeyDown}
        onChange={(event) => updateValue(event.target.value)}
        onPaste={(event) => {
          event.preventDefault()
          updateValue(event.clipboardData.getData("text"))
          onPaste?.(event)
        }}
        onBlur={(event) => {
          onValueChange(normalizeCurrencyValue(event.target.value).replace(/\.$/, ""))
          onBlur?.(event)
        }}
      />
    </div>
  )
}

export { CurrencyInput, formatCurrencyValue, normalizeCurrencyValue }
