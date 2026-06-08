import { useState } from "react";

import { AgendaView } from "@/components/taskify/agenda-view";
import { Header } from "@/components/taskify/header";
import { KanbanBoard } from "@/components/taskify/kanban-board";
import { MobileTaskList } from "@/components/taskify/mobile-task-list";
import { Sidebar } from "@/components/taskify/sidebar";

type ActiveView = "kanban" | "agenda";

export function TaskifyDashboard() {
  const [activeView, setActiveView] = useState<ActiveView>("kanban");

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
              <KanbanBoard />
            </div>
          </>
        ) : (
          <AgendaView />
        )}
      </div>
    </div>
  );
}
