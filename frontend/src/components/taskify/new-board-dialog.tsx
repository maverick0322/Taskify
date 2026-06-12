"use client"

import { type FormEvent, useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { getFriendlyErrorMessage } from "@/services/api"
import { createBoard } from "@/services/boardService"

interface NewBoardDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function NewBoardDialog({ open, onOpenChange }: NewBoardDialogProps) {
  const queryClient = useQueryClient()
  const [name, setName] = useState("")
  const [errorMessage, setErrorMessage] = useState("")

  const mutation = useMutation({
    mutationFn: createBoard,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["boards"] })
      reset()
      onOpenChange(false)
    },
    onError: (error) => {
      setErrorMessage(
        getFriendlyErrorMessage(
          error,
          "No pudimos crear el tablero. Intentalo de nuevo.",
        ),
      )
    },
  })

  function reset() {
    setName("")
    setErrorMessage("")
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const trimmedName = name.trim()
    if (!trimmedName) {
      setErrorMessage("Escribe un nombre para el tablero.")
      return
    }

    mutation.mutate(trimmedName)
  }

  function handleCancel() {
    if (mutation.isPending) {
      return
    }

    reset()
    onOpenChange(false)
  }

  function handleOpenChange(nextOpen: boolean) {
    if (mutation.isPending) {
      return
    }

    if (!nextOpen) {
      reset()
    }
    onOpenChange(nextOpen)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Nuevo Tablero</DialogTitle>
          <DialogDescription>
            Crea un espacio para organizar tareas, columnas y prioridades.
          </DialogDescription>
        </DialogHeader>

        <form className="flex flex-col gap-4 py-2" onSubmit={handleSubmit}>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="board-name">Nombre</Label>
            <Input
              id="board-name"
              placeholder="Ej. Lanzamiento Q3"
              value={name}
              onChange={(event) => setName(event.target.value)}
              disabled={mutation.isPending}
              autoFocus
            />
          </div>

          {errorMessage ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm font-medium text-red-700">
              {errorMessage}
            </p>
          ) : null}

          <DialogFooter className="gap-2 sm:gap-2">
            <Button
              variant="outline"
              type="button"
              onClick={handleCancel}
              disabled={mutation.isPending}
            >
              Cancelar
            </Button>
            <Button type="submit" disabled={!name.trim() || mutation.isPending}>
              {mutation.isPending ? "Creando..." : "Crear"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
