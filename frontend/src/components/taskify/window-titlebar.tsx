"use client"

import { Maximize2, Minus, X } from "lucide-react"
import type { MouseEvent } from "react"

import { Button } from "@/components/ui/button"
import { isTauriRuntime } from "@/lib/runtime"

export function WindowTitlebar() {
  if (!isTauriRuntime()) {
    return null
  }

  function handleWindowAction(
    event: MouseEvent<HTMLButtonElement>,
    action: "minimize" | "toggleMaximize" | "close",
  ) {
    event.preventDefault()
    event.stopPropagation()
    void import("@tauri-apps/api/window").then(({ getCurrentWindow }) => {
      const appWindow = getCurrentWindow()
      return appWindow[action]()
    })
  }

  return (
    <div
      data-tauri-drag-region
      className="flex h-8 shrink-0 select-none items-center justify-between border-b border-border bg-card pl-3 text-card-foreground"
    >
      <div
        data-tauri-drag-region
        className="flex min-w-0 items-center gap-2 text-xs font-semibold tracking-wide text-muted-foreground"
      >
        <span
          data-tauri-drag-region
          className="size-2 rounded-full bg-primary"
          aria-hidden="true"
        />
        <span data-tauri-drag-region className="truncate">
          Taskify
        </span>
      </div>

      <div className="flex h-full items-center">
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-11 rounded-none text-muted-foreground hover:bg-muted hover:text-foreground"
          aria-label="Minimizar ventana"
          onClick={(event) => handleWindowAction(event, "minimize")}
        >
          <Minus className="size-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-11 rounded-none text-muted-foreground hover:bg-muted hover:text-foreground"
          aria-label="Maximizar o restaurar ventana"
          onClick={(event) => handleWindowAction(event, "toggleMaximize")}
        >
          <Maximize2 className="size-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-11 rounded-none text-muted-foreground hover:bg-red-500 hover:text-white"
          aria-label="Cerrar ventana"
          onClick={(event) => handleWindowAction(event, "close")}
        >
          <X className="size-4" />
        </Button>
      </div>
    </div>
  )
}
