import { isTauriRuntime } from "@/lib/runtime"
import type { AppNotification } from "@/services/notification_api"

export async function notifyCriticalAlerts(count: number) {
  if (count <= 0) {
    return
  }

  try {
    const permissionGranted = await ensureNotificationPermission()
    if (!permissionGranted) {
      return
    }

    await sendTaskifyNotification(
      "Taskify",
      `Tienes ${count} alerta${count === 1 ? "" : "s"} crítica${
        count === 1 ? "" : "s"
      } hoy`,
    )
  } catch {
    // Native notifications are only available inside Tauri and may be denied by the OS.
  }
}

export async function notifyAppNotification(notification: AppNotification) {
  try {
    const permissionGranted = await ensureNotificationPermission()
    if (!permissionGranted) {
      return
    }

    await sendTaskifyNotification(notification.title, notification.message)
  } catch {
    // Native notifications are only available inside Tauri and may be denied by the OS.
  }
}

async function ensureNotificationPermission() {
  if (isTauriRuntime()) {
    const { isPermissionGranted, requestPermission } = await import(
      "@tauri-apps/plugin-notification"
    )
    if (await isPermissionGranted()) {
      return true
    }

    const permission = await requestPermission()
    return permission === "granted"
  }

  if (!("Notification" in window)) {
    return false
  }

  if (Notification.permission === "granted") {
    return true
  }

  const permission = await Notification.requestPermission()
  return permission === "granted"
}

async function sendTaskifyNotification(title: string, body: string) {
  if (isTauriRuntime()) {
    const { sendNotification } = await import("@tauri-apps/plugin-notification")
    sendNotification({ title, body })
    return
  }

  if ("Notification" in window && Notification.permission === "granted") {
    new Notification(title, { body })
  }
}
