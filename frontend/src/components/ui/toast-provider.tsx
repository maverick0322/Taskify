"use client"

import React, { createContext, useCallback, useContext, useMemo, useState } from "react"
import { cn } from "@/lib/utils"

type ToastVariant = "success" | "error"

interface Toast {
  id: string
  message: string
  variant: ToastVariant
}

interface ToastContextValue {
  success: (message: string) => void
  error: (message: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const pushToast = useCallback((message: string, variant: ToastVariant) => {
    const id = crypto.randomUUID()
    setToasts((current) => [...current, { id, message, variant }])
    window.setTimeout(() => {
      setToasts((current) => current.filter((toast) => toast.id !== id))
    }, 3500)
  }, [])

  const value = useMemo(
    () => ({
      success: (message: string) => pushToast(message, "success"),
      error: (message: string) => pushToast(message, "error"),
    }),
    [pushToast],
  )

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="pointer-events-none fixed right-4 top-16 z-[100] flex w-[calc(100vw-2rem)] max-w-sm flex-col gap-2 sm:right-6">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            role="status"
            className={cn(
              "rounded-lg border bg-background px-4 py-3 text-sm font-medium text-foreground shadow-lg",
              toast.variant === "success"
                ? "border-emerald-500/30"
                : "border-destructive/40",
            )}
          >
            {toast.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast() {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error("useToast must be used within ToastProvider")
  }
  return context
}
