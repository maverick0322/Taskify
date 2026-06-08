import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { AgendaView } from "@/components/taskify/agenda-view";
import { Header } from "@/components/taskify/header";
import { KanbanBoard } from "@/components/taskify/kanban-board";
import { MobileTaskList } from "@/components/taskify/mobile-task-list";
import { Sidebar } from "@/components/taskify/sidebar";
import { getTasks } from "@/services/taskService";

type ActiveView = "kanban" | "agenda";

export function TaskifyDashboard() {
  const [activeView, setActiveView] = useState<ActiveView>("kanban");
  const {
    data: tasks = [],
    isLoading,
    isError,
    error,
  } = useQuery({ queryKey: ["tasks"], queryFn: getTasks });

  const taskErrorMessage =
    error instanceof Error ? error.message : "No se pudo cargar el tablero";

  return (
    <div className="flex h-screen w-full overflow-hidden bg-canvas">
      <div className="hidden md:flex md:shrink-0">
        <Sidebar
          className="h-full"
          activeView={activeView}
          onViewChange={setActiveView}
        />
      </div>

      <div className="flex flex-1 flex-col overflow-hidden">
        <Header activeView={activeView} onViewChange={setActiveView} />

        {activeView === "kanban" ? (
          <>
            <div className="flex flex-1 flex-col overflow-hidden md:hidden">
              <MobileTaskList />
            </div>

            <div className="hidden flex-1 overflow-hidden md:flex">
              {isLoading ? (
                <div className="flex flex-1 items-center justify-center bg-canvas text-sm font-medium text-muted-foreground">
                  Cargando tablero...
                </div>
              ) : isError ? (
                <div className="flex flex-1 items-center justify-center bg-canvas px-6 text-center text-sm font-medium text-red-600">
                  {taskErrorMessage}
                </div>
              ) : (
                <KanbanBoard tasks={tasks} />
              )}
            </div>
          </>
        ) : (
          <AgendaView />
        )}
      </div>
    </div>
  );
}
