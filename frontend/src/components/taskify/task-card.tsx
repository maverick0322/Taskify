"use client"

import { Draggable } from "@hello-pangea/dnd"
import type { CSSProperties } from "react"

import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Clock, Paperclip, MessageSquare, MoreHorizontal } from "lucide-react"
import { Button } from "@/components/ui/button"

type Priority = "Alta" | "Media" | "Baja"

interface TaskCardProps {
  id: string
  index: number
  title: string
  description?: string
  priority: Priority
  dueDate: string
  tag?: string
  assignees?: { name: string; seed: string }[]
  comments?: number
  attachments?: number
}

const priorityConfig: Record<Priority, { label: string; className: string }> = {
  Alta: {
    label: "Alta",
    className:
      "bg-red-100 text-red-700 border-red-200 dark:bg-red-950/50 dark:text-red-400 dark:border-red-900",
  },
  Media: {
    label: "Media",
    className:
      "bg-amber-100 text-amber-700 border-amber-200 dark:bg-amber-950/50 dark:text-amber-400 dark:border-amber-900",
  },
  Baja: {
    label: "Baja",
    className:
      "bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-950/50 dark:text-blue-400 dark:border-blue-900",
  },
}

export function TaskCard({
  id,
  index,
  title,
  description,
  priority,
  dueDate = "Sin fecha",
  tag,
  assignees = [],
  comments = 0,
  attachments = 0,
}: TaskCardProps) {
  const { label, className } = priorityConfig[priority]

  return (
    <Draggable draggableId={id} index={index}>
      {(provided) => {
        const draggableStyle = provided.draggableProps.style as CSSProperties

        return (
        <article
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          style={draggableStyle}
          className="group relative rounded-xl border border-border bg-card p-4 shadow-sm transition-all hover:shadow-md hover:-translate-y-0.5 hover:border-primary/30 cursor-grab active:cursor-grabbing"
        >
          {/* Header row */}
          <div className="mb-3 flex items-start justify-between gap-2">
            <div className="flex flex-wrap gap-1.5">
              <Badge
                variant="outline"
                className={cn("text-[11px] font-semibold px-2 py-0.5 rounded-full border", className)}
              >
                {label}
              </Badge>
              {tag && (
                <Badge
                  variant="secondary"
                  className="text-[11px] px-2 py-0.5 rounded-full"
                >
                  {tag}
                </Badge>
              )}
            </div>
            <Button
              variant="ghost"
              size="icon"
              className="size-6 shrink-0 opacity-0 group-hover:opacity-100 text-muted-foreground transition-opacity -mt-0.5 -mr-1"
              aria-label="Más opciones"
            >
              <MoreHorizontal className="size-3.5" />
            </Button>
          </div>

          {/* Title */}
          <h3 className="mb-1.5 text-sm font-semibold leading-snug text-card-foreground">
            {title}
          </h3>

          {/* Description */}
          {description && (
            <p className="mb-3 text-xs leading-relaxed text-muted-foreground line-clamp-2">
              {description}
            </p>
          )}

          {/* Footer */}
          <div className="flex items-center justify-between pt-3 border-t border-border/60">
            {/* Date */}
            <div className="flex items-center gap-1 text-muted-foreground">
              <Clock className="size-3" />
              <span className="text-[11px] font-medium">{dueDate}</span>
            </div>

            <div className="flex items-center gap-2">
              {/* Comments / Attachments */}
              {comments > 0 && (
                <div className="flex items-center gap-1 text-muted-foreground">
                  <MessageSquare className="size-3" />
                  <span className="text-[11px]">{comments}</span>
                </div>
              )}
              {attachments > 0 && (
                <div className="flex items-center gap-1 text-muted-foreground">
                  <Paperclip className="size-3" />
                  <span className="text-[11px]">{attachments}</span>
                </div>
              )}

              {/* Assignee Avatars */}
              {assignees.length > 0 && (
                <div className="flex -space-x-1.5">
                  {assignees.slice(0, 3).map((a) => (
                    <Avatar key={a.seed} className="size-5 ring-1 ring-card">
                      <AvatarImage
                        src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${a.seed}`}
                        alt={a.name}
                      />
                      <AvatarFallback className="bg-primary/10 text-primary text-[9px] font-bold">
                        {a.name.charAt(0)}
                      </AvatarFallback>
                    </Avatar>
                  ))}
                </div>
              )}
            </div>
          </div>
        </article>
        )
      }}
    </Draggable>
  )
}
