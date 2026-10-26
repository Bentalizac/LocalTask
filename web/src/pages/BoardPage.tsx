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
  const { tickets, error, setStatus, create, archiveDone } = useTickets(project);
  const [archiving, setArchiving] = useState(false);
  const [archiveError, setArchiveError] = useState<string | null>(null);
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

  // Archived tickets are cleared from the board; their status stays done.
  const visible = tickets.filter((t) => !t.archived_at);

  async function handleArchiveDone(count: number) {
    if (!window.confirm(`Archive ${count} done ticket${count === 1 ? "" : "s"}? They leave the board but stay done, and List can still show them.`)) return;
    setArchiving(true);
    setArchiveError(null);
    try {
      await archiveDone();
    } catch (err) {
      setArchiveError(err instanceof Error ? err.message : "Failed to archive tickets");
    } finally {
      setArchiving(false);
    }
  }

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

      {archiveError && <div className="error-banner">{archiveError}</div>}

      <div className="board">
        {BOARD_COLUMNS.map((status) => {
          const columnTickets = visible.filter((t) => t.status === status);
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
                <span className="board-column-meta">
                  {status === "done" && columnTickets.length > 0 && (
                    <button
                      className="btn btn-sm board-column-action"
                      onClick={() => handleArchiveDone(columnTickets.length)}
                      disabled={archiving}
                      title="Clear every done ticket from the board"
                    >
                      {archiving ? "Archiving…" : "Archive done"}
                    </button>
                  )}
                  <span className="board-column-count">
                    {columnTickets.length}
                  </span>
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
