import { useEffect } from "react"
import { useTheme } from "@/components/theme-provider"
import type { CurrentView } from "@/components/taskify/navigation"
import { useUIStore } from "@/store/useUIStore"

interface UseKeyboardShortcutsOptions {
  setCurrentView: (view: CurrentView) => void
}

export function useKeyboardShortcuts({
  setCurrentView,
}: UseKeyboardShortcutsOptions) {
  const { resolvedTheme, setTheme } = useTheme()

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        event.preventDefault()
        useUIStore.getState().closeGlobalOverlays()
        return
      }

      if (!event.ctrlKey || event.altKey || event.metaKey || isEditableTarget(event.target)) {
        return
      }

      const key = event.key.toLowerCase()
      const uiStore = useUIStore.getState()

      if (key === "n") {
        event.preventDefault()
        uiStore.toggleNewTaskModal()
        return
      }

      if (key === "m") {
        event.preventDefault()
        setCurrentView("financial")
        uiStore.toggleNewMovementModal()
        return
      }

      if (key === "k") {
        event.preventDefault()
        uiStore.toggleHelpModal()
        return
      }

      if (key === "j") {
        event.preventDefault()
        setTheme(resolvedTheme === "dark" ? "light" : "dark")
        return
      }

      if (key === "i") {
        event.preventDefault()
        uiStore.toggleNotifications()
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => {
      window.removeEventListener("keydown", handleKeyDown)
    }
  }, [resolvedTheme, setCurrentView, setTheme])
}

function isEditableTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) {
    return false
  }

  const tagName = target.tagName.toLowerCase()
  return (
    tagName === "input" ||
    tagName === "textarea" ||
    tagName === "select" ||
    target.isContentEditable
  )
}
