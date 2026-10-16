import { useState, type CSSProperties, type DragEvent } from "react";
import { useParams } from "react-router-dom";
import { useTickets } from "../hooks/useTickets";
import { useJiraBaseUrl } from "../hooks/useJiraBaseUrl";
import { TicketCard } from "../components/TicketCard";
import { NewTicketModal } from "../components/NewTicketModal";
import { BOARD_COLUMNS } from "../types";
import type { TicketStatus } from "../types";

export function BoardPage() {
  const { project = "" } = useParams<{ project: string }>();
  const { tickets, error, setStatus, create } = useTickets(project);
  const jiraBaseUrl = useJiraBaseUrl(project);
  const [showNew, setShowNew] = useState(false);
  const [dragOverColumn, setDragOverColumn] = useState<TicketStatus | null>(
    null,
  );

  if (error)
    return (
      <div className="page">
        <div className="error-banner">{error}</div>
      </div>
    );
  if (!tickets)
    return (
      <div className="page">
        <div className="loading">Loading…</div>
      </div>
    );

  function handleDrop(e: DragEvent<HTMLDivElement>, status: TicketStatus) {
    e.preventDefault();
    setDragOverColumn(null);
    const id = e.dataTransfer.getData("text/plain");
    if (id) setStatus(id, status);
  }

  return (
    <div className="page page-board">
      <div className="page-header">
        <h1>Board</h1>
        <button className="btn btn-primary" onClick={() => setShowNew(true)}>
          + New task
        </button>
      </div>

      <div className="board">
        {BOARD_COLUMNS.map((status) => {
          const columnTickets = tickets.filter((t) => t.status === status);
          const style = {
            "--status-color": `var(--status-${status})`,
          } as CSSProperties;
          return (
            <div
              className={`board-column${dragOverColumn === status ? " drag-over" : ""}`}
              key={status}
              onDragOver={(e) => {
                e.preventDefault();
                if (dragOverColumn !== status) setDragOverColumn(status);
              }}
              onDragLeave={(e) => {
                if (!e.currentTarget.contains(e.relatedTarget as Node)) {
                  setDragOverColumn((cur) => (cur === status ? null : cur));
                }
              }}
              onDrop={(e) => handleDrop(e, status)}
            >
              <div className="board-column-header" style={style}>
                <span className="board-column-title">
                  {status.replace("_", " ")}
                </span>
                <span className="board-column-count">
                  {columnTickets.length}
                </span>
              </div>
              <div className="board-column-cards">
                {columnTickets.map((t) => (
                  <TicketCard
                    key={t.id}
                    project={project}
                    ticket={t}
                    jiraBaseUrl={jiraBaseUrl}
                    onStatusChange={setStatus}
                  />
                ))}
              </div>
            </div>
          );
        })}
      </div>

      {showNew && (
        <NewTicketModal onClose={() => setShowNew(false)} onCreate={create} />
      )}
    </div>
  );
}
