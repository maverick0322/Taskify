import {
  isPermissionGranted,
  requestPermission,
  sendNotification,
} from "@tauri-apps/plugin-notification"

export async function notifyCriticalAlerts(count: number) {
  if (count <= 0) {
    return
  }

  try {
    const permissionGranted = await ensureNotificationPermission()
    if (!permissionGranted) {
      return
    }

    sendNotification({
      title: "Taskify",
      body: `Tienes ${count} alerta${count === 1 ? "" : "s"} crítica${
        count === 1 ? "" : "s"
      } hoy`,
    })
  } catch {
    // Native notifications are only available inside Tauri and may be denied by the OS.
  }
}

async function ensureNotificationPermission() {
  if (await isPermissionGranted()) {
    return true
  }

  const permission = await requestPermission()
  return permission === "granted"
}
