import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Bot, LayoutDashboard } from "lucide-react";

import { AgendaView } from "@/components/taskify/agenda-view";
import { EmptyState } from "@/components/taskify/empty-state";
import { Header } from "@/components/taskify/header";
import { KanbanBoard } from "@/components/taskify/kanban-board";
import { MobileTaskList } from "@/components/taskify/mobile-task-list";
import type { CurrentView } from "@/components/taskify/navigation";
import { Sidebar } from "@/components/taskify/sidebar";
import { getBoards } from "@/services/boardService";
import { getTasks } from "@/services/taskService";

export function TaskifyDashboard() {
  const [currentView, setCurrentView] = useState<CurrentView>("tasks");
  const [selectedBoardId, setSelectedBoardId] = useState<string>();
  const {
    data: boards = [],
    isLoading: boardsLoading,
    isError: boardsIsError,
    error: boardsError,
  } = useQuery({ queryKey: ["boards"], queryFn: getBoards });
  const {
    data: tasks = [],
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: ["tasks", selectedBoardId],
    queryFn: () => getTasks(selectedBoardId),
    enabled: !!selectedBoardId,
  });

  const taskErrorMessage =
    error instanceof Error ? error.message : "No se pudo cargar el tablero";
  const boardsErrorMessage =
    boardsError instanceof Error
      ? boardsError.message
      : "No se pudieron cargar los tableros";
  const selectedBoard = boards.find((board) => board.id === selectedBoardId);

  useEffect(() => {
    if (boards.length === 0) {
      setSelectedBoardId(undefined);
      return;
    }

    const selectedBoardStillExists = boards.some(
      (board) => board.id === selectedBoardId,
    );

    if (!selectedBoardId || !selectedBoardStillExists) {
      setSelectedBoardId(boards[0].id);
    }
  }, [boards, selectedBoardId]);

  return (
    <div className="flex h-screen w-full overflow-hidden bg-canvas">
      <div className="hidden md:flex md:shrink-0">
        <Sidebar
          className="h-full"
          activeView={currentView}
          boards={boards}
          boardsError={boardsIsError ? boardsErrorMessage : undefined}
          boardsLoading={boardsLoading}
          onViewChange={setCurrentView}
          selectedBoardId={selectedBoardId}
          onBoardSelect={(board) => setSelectedBoardId(board.id)}
        />
      </div>

      <div className="flex flex-1 flex-col overflow-hidden">
        <Header
          activeView={currentView}
          boards={boards}
          boardsError={boardsIsError ? boardsErrorMessage : undefined}
          boardsLoading={boardsLoading}
          onViewChange={setCurrentView}
          selectedBoardId={selectedBoardId}
          selectedBoardName={selectedBoard?.name}
          onBoardSelect={(board) => setSelectedBoardId(board.id)}
        />

        {currentView === "tasks" ? (
          <>
            <div className="flex flex-1 flex-col overflow-hidden md:hidden">
              {selectedBoardId ? (
                <MobileTaskList />
              ) : (
                <div className="flex flex-1 items-center justify-center bg-canvas px-6 text-center text-sm font-medium text-muted-foreground">
                  Selecciona un tablero del menú lateral para ver sus tareas
                </div>
              )}
            </div>

            <div className="hidden flex-1 overflow-hidden md:flex">
              {!selectedBoardId ? (
                <div className="flex flex-1 items-center justify-center bg-canvas px-6 text-center text-sm font-medium text-muted-foreground">
                  Selecciona un tablero del menú lateral para ver sus tareas
                </div>
              ) : isLoading ? (
                <div className="flex flex-1 items-center justify-center bg-canvas text-sm font-medium text-muted-foreground">
                  Cargando tablero...
                </div>
              ) : isError ? (
                <div className="flex flex-1 items-center justify-center bg-canvas px-6 text-center text-sm font-medium text-red-600">
                  {taskErrorMessage}
                </div>
              ) : (
                <KanbanBoard selectedBoardId={selectedBoardId} tasks={tasks} />
              )}
            </div>
          </>
        ) : null}

        {currentView === "agenda" ? (
          <AgendaView selectedBoardId={selectedBoardId} />
        ) : null}

        {currentView === "dashboard" ? (
          <EmptyState icon={LayoutDashboard} title="Panel de Control" />
        ) : null}

        {currentView === "automations" ? (
          <EmptyState icon={Bot} title="Automatizaciones" />
        ) : null}
      </div>
    </div>
  );
}
