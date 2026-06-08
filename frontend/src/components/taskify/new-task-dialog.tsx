"use client"

import { useState } from "react"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { Calendar } from "@/components/ui/calendar"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { CalendarIcon } from "lucide-react"
import { cn } from "@/lib/utils"

// Month names for formatting the selected date label
const MONTH_NAMES = [
  "Ene","Feb","Mar","Abr","May","Jun",
  "Jul","Ago","Sep","Oct","Nov","Dic",
]

function formatDate(date: Date): string {
  return `${date.getDate()} ${MONTH_NAMES[date.getMonth()]} ${date.getFullYear()}`
}

interface NewTaskDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function NewTaskDialog({ open, onOpenChange }: NewTaskDialogProps) {
  const [title,       setTitle]       = useState("")
  const [description, setDescription] = useState("")
  const [priority,    setPriority]    = useState("")
  const [dueDate,     setDueDate]     = useState<Date | undefined>(undefined)
  const [time,        setTime]        = useState("")
  const [calOpen,     setCalOpen]     = useState(false)

  function reset() {
    setTitle("")
    setDescription("")
    setPriority("")
    setDueDate(undefined)
    setTime("")
    setCalOpen(false)
  }

  function handleSave() {
    reset()
    onOpenChange(false)
  }

  function handleCancel() {
    reset()
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Nueva Tarea</DialogTitle>
          <DialogDescription>
            Completa los datos para agregar una nueva tarea al tablero.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4 py-2">
          {/* Title */}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="task-title">Título</Label>
            <Input
              id="task-title"
              placeholder="Ej. Diseñar pantalla de inicio"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
          </div>

          {/* Description */}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="task-description">Descripción</Label>
            <Textarea
              id="task-description"
              placeholder="Describe brevemente la tarea..."
              className="min-h-[80px] resize-none"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          {/* Priority */}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="task-priority">Prioridad</Label>
            <Select value={priority} onValueChange={setPriority}>
              <SelectTrigger id="task-priority">
                <SelectValue placeholder="Selecciona una prioridad" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="Alta">Alta</SelectItem>
                <SelectItem value="Media">Media</SelectItem>
                <SelectItem value="Baja">Baja</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Due date + time — side by side */}
          <div className="grid grid-cols-2 gap-3">
            {/* Date picker */}
            <div className="flex flex-col gap-1.5">
              <Label>Fecha de entrega</Label>
              <Popover open={calOpen} onOpenChange={setCalOpen}>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    className={cn(
                      "w-full justify-start gap-2 text-left font-normal",
                      !dueDate && "text-muted-foreground"
                    )}
                  >
                    <CalendarIcon className="size-4 shrink-0" />
                    <span className="truncate text-sm">
                      {dueDate ? formatDate(dueDate) : "Seleccionar"}
                    </span>
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={dueDate}
                    onSelect={(date) => {
                      setDueDate(date)
                      setCalOpen(false)
                    }}
                  />
                </PopoverContent>
              </Popover>
            </div>

            {/* Time */}
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="task-time">Hora</Label>
              <Input
                id="task-time"
                type="time"
                value={time}
                onChange={(e) => setTime(e.target.value)}
                className="w-full"
              />
            </div>
          </div>
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button variant="outline" onClick={handleCancel}>
            Cancelar
          </Button>
          <Button onClick={handleSave} disabled={!title.trim()}>
            Guardar
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
