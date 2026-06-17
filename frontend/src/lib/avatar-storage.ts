import { convertFileSrc } from "@tauri-apps/api/core"
import { isTauriRuntime } from "@/lib/runtime"

export function localAvatarSrc(path?: string | null) {
  if (path === "") {
    console.error("avatar local path is empty")
    return undefined
  }
  if (!path) {
    return undefined
  }

  if (!isTauriRuntime()) {
    return undefined
  }

  if (!isAbsolutePath(path)) {
    console.error("avatar local path is not absolute", path)
    return undefined
  }

  try {
    return convertFileSrc(path)
  } catch (error) {
    console.error("failed to convert avatar local path", error)
    return undefined
  }
}

export async function selectAndStoreAvatar(userId: string) {
  if (!isTauriRuntime()) {
    throw new Error("La selección de avatar local solo está disponible en la app de escritorio.")
  }

  const [{ open }, { appDataDir, join }, { BaseDirectory, copyFile, exists, mkdir }] =
    await Promise.all([
      import("@tauri-apps/plugin-dialog"),
      import("@tauri-apps/api/path"),
      import("@tauri-apps/plugin-fs"),
    ])

  const selectedPath = await open({
    multiple: false,
    directory: false,
    filters: [
      {
        name: "Imagen",
        extensions: ["png", "jpg", "jpeg", "webp"],
      },
    ],
  })

  if (typeof selectedPath !== "string") {
    return null
  }

  const extension = extensionFromPath(selectedPath) ?? "jpg"
  const avatarFileName = `${userId}.${extension}`
  const avatarsDirectory = await join(await appDataDir(), "avatars")
  const avatarDestination = await join(avatarsDirectory, avatarFileName)

  if (!(await exists("avatars", { baseDir: BaseDirectory.AppData }))) {
    await mkdir("avatars", { baseDir: BaseDirectory.AppData, recursive: true })
  }

  await copyFile(selectedPath, avatarDestination)

  return avatarDestination
}


export function isAbsolutePath(path: string) {
  return /^[a-zA-Z]:[\\/]/.test(path) || path.startsWith("/") || path.startsWith("\\\\")
}

function extensionFromPath(path: string) {
  const normalizedPath = path.replace(/\\/g, "/")
  const fileName = normalizedPath.split("/").pop()
  const extension = fileName?.split(".").pop()?.toLowerCase()

  if (!extension || extension === fileName) {
    return null
  }

  return extension
}
