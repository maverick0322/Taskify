"use client"

import React from "react"
import { Zap } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useUIStore } from "@/store/useUIStore"

const shortcuts = [
  { keys: ["Ctrl", "N"], label: "Nueva tarea" },
  { keys: ["Ctrl", "M"], label: "Nuevo movimiento financiero" },
  { keys: ["Ctrl", "K"], label: "Acerca de / Ayuda" },
  { keys: ["Ctrl", "J"], label: "Cambiar modo claro/oscuro" },
  { keys: ["Ctrl", "I"], label: "Ver notificaciones pendientes" },
  { keys: ["Esc"], label: "Cerrar modal activo" },
]

export function HelpDialog() {
  const open = useUIStore((state) => state.isHelpModalOpen)
  const setOpen = useUIStore((state) => state.setHelpModalOpen)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Acerca de Taskify</DialogTitle>
          <DialogDescription>
            Organización, agenda y finanzas en una sola app.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          <div className="flex items-center gap-4 rounded-lg border border-border bg-muted/40 p-4">
            <div className="flex size-12 shrink-0 items-center justify-center rounded-xl bg-primary">
              <Zap className="size-6 text-primary-foreground" strokeWidth={2.5} />
            </div>
            <div>
              <p className="text-2xl font-bold tracking-tight text-foreground">
                Taskify
              </p>
              <p className="text-sm text-muted-foreground">Versión 1.0.0</p>
            </div>
          </div>

          <section className="space-y-3">
            <h3 className="text-sm font-semibold text-foreground">
              Atajos de Teclado
            </h3>
            <div className="space-y-2">
              {shortcuts.map((shortcut) => (
                <KeyboardShortcut
                  key={shortcut.label}
                  keys={shortcut.keys}
                  label={shortcut.label}
                />
              ))}
            </div>
          </section>
        </div>

        <DialogFooter>
          <Button variant="ghost" asChild>
            <a href="mailto:support@taskify.local">Contactar Soporte</a>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function KeyboardShortcut({
  keys,
  label,
}: {
  keys: string[]
  label: string
}) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-md border border-border/70 px-3 py-2">
      <div className="flex items-center gap-1.5">
        {keys.map((key, index) => (
          <React.Fragment key={key}>
            {index > 0 ? (
              <span className="text-xs text-muted-foreground">+</span>
            ) : null}
            <kbd className="rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-[11px] font-semibold text-foreground shadow-sm">
              {key}
            </kbd>
          </React.Fragment>
        ))}
      </div>
      <span className="text-sm text-muted-foreground">{label}</span>
    </div>
  )
}
