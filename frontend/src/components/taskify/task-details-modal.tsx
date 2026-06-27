"use client"

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { cn } from "@/lib/utils"
import { CalendarDays, MessageSquare, Paperclip, Pencil, Trash2, Users } from "lucide-react"

interface TaskDetailsModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  priorityLabel: string
  priorityClassName: string
  dueDate: string
  tag?: string
  assignees: { name: string; seed: string }[]
  comments: number
  attachments: number
  deletePending?: boolean
  onEdit: () => void
  onDelete: () => void
}

export function TaskDetailsModal({
  open,
  onOpenChange,
  title,
  description,
  priorityLabel,
  priorityClassName,
  dueDate,
  tag,
  assignees,
  comments,
  attachments,
  deletePending = false,
  onEdit,
  onDelete,
}: TaskDetailsModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader className="pr-8">
          <div className="flex flex-wrap gap-1.5">
            <Badge
              variant="outline"
              className={cn(
                "rounded-full border px-2 py-0.5 text-[11px] font-semibold",
                priorityClassName,
              )}
            >
              {priorityLabel}
            </Badge>
            {tag ? (
              <Badge
                variant="secondary"
                className="rounded-full px-2 py-0.5 text-[11px]"
              >
                {tag}
              </Badge>
            ) : null}
          </div>
          <DialogTitle className="text-lg leading-snug">{title}</DialogTitle>
          <DialogDescription>
            Detalles de la tarea seleccionada.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-1">
          <section className="grid gap-2">
            <h4 className="text-xs font-semibold uppercase text-muted-foreground">
              Descripción
            </h4>
            <p className="whitespace-pre-wrap rounded-lg border border-border/70 bg-muted/30 p-3 text-sm leading-relaxed text-foreground">
              {description?.trim() || "Sin descripción"}
            </p>
          </section>

          <div className="grid gap-3 sm:grid-cols-3">
            <div className="rounded-lg border border-border/70 p-3">
              <div className="mb-1 flex items-center gap-1.5 text-muted-foreground">
                <CalendarDays className="size-3.5" />
                <span className="text-xs font-medium">Fecha</span>
              </div>
              <p className="text-sm font-semibold">{dueDate}</p>
            </div>
            <div className="rounded-lg border border-border/70 p-3">
              <div className="mb-1 flex items-center gap-1.5 text-muted-foreground">
                <MessageSquare className="size-3.5" />
                <span className="text-xs font-medium">Comentarios</span>
              </div>
              <p className="text-sm font-semibold">{comments}</p>
            </div>
            <div className="rounded-lg border border-border/70 p-3">
              <div className="mb-1 flex items-center gap-1.5 text-muted-foreground">
                <Paperclip className="size-3.5" />
                <span className="text-xs font-medium">Adjuntos</span>
              </div>
              <p className="text-sm font-semibold">{attachments}</p>
            </div>
          </div>

          <section className="grid gap-2">
            <div className="flex items-center gap-1.5 text-muted-foreground">
              <Users className="size-3.5" />
              <h4 className="text-xs font-semibold uppercase">Asignados</h4>
            </div>
            {assignees.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {assignees.map((assignee) => (
                  <div
                    key={assignee.seed}
                    className="flex items-center gap-2 rounded-full border border-border/70 px-2 py-1"
                  >
                    <Avatar className="size-6">
                      <AvatarImage
                        src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${assignee.seed}`}
                        alt={assignee.name}
                      />
                      <AvatarFallback className="bg-primary/10 text-[10px] font-bold text-primary">
                        {assignee.name.charAt(0)}
                      </AvatarFallback>
                    </Avatar>
                    <span className="text-sm font-medium">{assignee.name}</span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">Sin asignados</p>
            )}
          </section>
        </div>

        <DialogFooter className="gap-2 sm:gap-2">
          <Button variant="outline" type="button" onClick={onEdit}>
            <Pencil className="size-4" />
            Editar
          </Button>
          <Button
            variant="destructive"
            type="button"
            disabled={deletePending}
            onClick={onDelete}
          >
            <Trash2 className="size-4" />
            Eliminar
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
