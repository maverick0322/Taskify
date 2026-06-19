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

  function stopTitlebarDrag(event: MouseEvent<HTMLButtonElement>) {
    event.stopPropagation()
  }

  function handleTitlebarMouseDown(event: MouseEvent<HTMLDivElement>) {
    if (event.button !== 0) {
      return
    }

    event.preventDefault()
    event.stopPropagation()
    void import("@tauri-apps/api/window").then(({ getCurrentWindow }) =>
      getCurrentWindow().startDragging(),
    )
  }

  return (
    <div
      data-tauri-drag-region
      className="flex h-8 shrink-0 select-none items-center border-b border-border bg-card text-card-foreground"
      onMouseDown={handleTitlebarMouseDown}
    >
      <div
        data-tauri-drag-region
        className="flex h-full min-w-0 flex-1 items-center gap-2 pl-3 text-xs font-semibold tracking-wide text-muted-foreground"
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

      <div className="flex h-full shrink-0 items-center">
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-11 rounded-none text-muted-foreground hover:bg-muted hover:text-foreground"
          aria-label="Minimizar ventana"
          onMouseDown={stopTitlebarDrag}
          onClick={(event) => handleWindowAction(event, "minimize")}
        >
          <Minus className="size-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-11 rounded-none text-muted-foreground hover:bg-muted hover:text-foreground"
          aria-label="Maximizar o restaurar ventana"
          onMouseDown={stopTitlebarDrag}
          onClick={(event) => handleWindowAction(event, "toggleMaximize")}
        >
          <Maximize2 className="size-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-11 rounded-none text-muted-foreground hover:bg-red-500 hover:text-white"
          aria-label="Cerrar ventana"
          onMouseDown={stopTitlebarDrag}
          onClick={(event) => handleWindowAction(event, "close")}
        >
          <X className="size-4" />
        </Button>
      </div>
    </div>
  )
}
