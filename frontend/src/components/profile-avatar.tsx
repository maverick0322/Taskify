"use client"

import { useEffect, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Camera, Loader2 } from "lucide-react"

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { localAvatarSrc, selectAndStoreAvatar } from "@/lib/avatar-storage"
import { isTauriRuntime } from "@/lib/runtime"
import { cn } from "@/lib/utils"
import { ApiError, getFriendlyErrorMessage } from "@/services/api"
import {
  getCurrentUserProfile,
  updateUserAvatar,
  type UserProfileResponse,
} from "@/services/userService"
import { useAuthStore, type AuthUser } from "@/store/useAuthStore"

interface ProfileAvatarProps {
  className?: string
  editable?: boolean
}

export function ProfileAvatar({ className, editable = true }: ProfileAvatarProps) {
  const queryClient = useQueryClient()
  const isDesktopRuntime = isTauriRuntime()
  const canEditAvatar = editable && isDesktopRuntime
  const user = useAuthStore((state) => state.user)
  const updateUserProfile = useAuthStore((state) => state.updateUserProfile)
  const [errorMessage, setErrorMessage] = useState("")
  const profileQuery = useQuery({
    queryKey: ["users", "me"],
    queryFn: getCurrentUserProfile,
    enabled: Boolean(user),
    staleTime: 5 * 60 * 1000,
  })
  const avatarMutation = useMutation({
    mutationFn: async () => {
      if (profileQuery.error instanceof ApiError && profileQuery.error.status === 401) {
        throw new ApiError(401, "Tu sesión expiró. Inicia sesión nuevamente.")
      }

      const userId = profileQuery.data?.id || user?.id
      if (!userId) {
        throw new Error("No hay usuario activo")
      }
      if (!isDesktopRuntime) {
        throw new Error("La carga de avatar solo está disponible en la app de escritorio.")
      }

      const avatarLocalPath = await selectAndStoreAvatar(userId)
      if (!avatarLocalPath) {
        return null
      }

      return updateUserAvatar(userId, avatarLocalPath)
    },
    onSuccess: async (profile) => {
      if (!profile) {
        return
      }
      updateUserProfile(profileToAuthUserPatch(profile))
      queryClient.setQueryData(["users", "me"], profile)
      await queryClient.invalidateQueries({ queryKey: ["users", "me"] })
      setErrorMessage("")
    },
    onError: (error) => {
      console.error("avatar update failed", error)
      setErrorMessage(avatarErrorMessage(error))
    },
  })

  useEffect(() => {
    if (profileQuery.data) {
      updateUserProfile(profileToAuthUserPatch(profileQuery.data))
    }
  }, [profileQuery.data, updateUserProfile])

  const activeUser = mergeUserAndProfile(user, profileQuery.data)
  const avatarSource = isDesktopRuntime
    ? localAvatarSrc(activeUser?.avatarLocalPath) ?? activeUser?.avatarUrl ?? undefined
    : undefined
  const isPending = avatarMutation.isPending || profileQuery.isLoading

  function handleClick() {
    if (!canEditAvatar || avatarMutation.isPending) {
      return
    }
    avatarMutation.mutate()
  }

  return (
    <div className="relative">
      <button
        type="button"
        className={cn(
          "group relative rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
          !canEditAvatar && "pointer-events-none",
        )}
        aria-label={canEditAvatar ? "Cambiar foto de perfil" : "Foto de perfil"}
        disabled={!canEditAvatar || avatarMutation.isPending}
        onClick={handleClick}
      >
        <Avatar className={className}>
          {avatarSource ? (
            <AvatarImage
              src={avatarSource}
              alt={activeUser?.fullName ?? "Taskify User"}
              onError={(event) => {
                console.error("avatar image failed to load", {
                  src: event.currentTarget.currentSrc || event.currentTarget.src,
                  avatarLocalPath: activeUser?.avatarLocalPath,
                  avatarUrl: activeUser?.avatarUrl,
                })
              }}
            />
          ) : null}
          <AvatarFallback className="bg-primary text-primary-foreground text-xs font-semibold">
            {activeUser?.initials ?? "TU"}
          </AvatarFallback>
        </Avatar>
        {canEditAvatar ? (
          <span className="absolute inset-0 flex items-center justify-center rounded-full bg-black/45 text-white opacity-0 transition-opacity group-hover:opacity-100">
            {isPending ? <Loader2 className="size-3.5 animate-spin" /> : <Camera className="size-3.5" />}
          </span>
        ) : null}
      </button>
      {errorMessage ? (
        <span
          className="absolute left-1/2 top-full z-20 mt-2 w-44 -translate-x-1/2 rounded-md bg-destructive px-2 py-1 text-center text-[11px] font-medium text-white shadow-lg"
          role="alert"
        >
          {errorMessage}
        </span>
      ) : null}
    </div>
  )
}

function avatarErrorMessage(error: unknown) {
  if (error instanceof ApiError && error.status === 401) {
    return "Tu sesión expiró. Inicia sesión nuevamente."
  }

  const message = error instanceof Error ? error.message.toLowerCase() : ""
  if (
    message.includes("file") ||
    message.includes("copy") ||
    message.includes("permission") ||
    message.includes("denied") ||
    message.includes("too large")
  ) {
    return "Archivo demasiado grande o no compatible"
  }

  return getFriendlyErrorMessage(error, "Error al guardar la imagen")
}

function profileToAuthUserPatch(profile: UserProfileResponse): Partial<AuthUser> {
  const fullName = [profile.firstName, profile.lastName].filter(Boolean).join(" ")
  return {
    id: profile.id,
    email: profile.email,
    firstName: profile.firstName,
    lastName: profile.lastName,
    fullName: fullName || profile.email || "Taskify User",
    initials: initialsFromName(profile.firstName, profile.lastName, profile.email),
    avatarLocalPath: profile.avatarLocalPath,
    avatarUrl: profile.avatarUrl,
  }
}

function mergeUserAndProfile(user: AuthUser | null, profile?: UserProfileResponse) {
  if (!user || !profile) {
    return user
  }

  return {
    ...user,
    ...profileToAuthUserPatch(profile),
  }
}

function initialsFromName(firstName: string, lastName: string, email: string) {
  const initials = `${firstName.charAt(0)}${lastName.charAt(0)}`.trim()
  if (initials) {
    return initials.toUpperCase()
  }

  return (email.charAt(0) || "T").toUpperCase()
}
