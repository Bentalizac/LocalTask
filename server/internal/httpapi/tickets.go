package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"localtask/internal/project"
	"localtask/internal/ticket"
)

func (s *Server) handleListTickets(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	filter, err := parseTicketFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	tickets := p.ListTickets(filter)
	if tickets == nil {
		tickets = []ticket.Ticket{}
	}
	writeJSON(w, http.StatusOK, tickets)
}

func (s *Server) handleGetTicket(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	t, err := p.GetTicket(r.PathValue("ticket"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	var req createTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	created, err := p.CreateTicket(req.toTicket())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if err := p.Save(); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateTicket(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	var req updateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	updated, err := p.UpdateTicket(r.PathValue("ticket"), req.toPatch())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if err := p.Save(); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func parseTicketFilter(r *http.Request) (project.TicketFilter, error) {
	q := r.URL.Query()
	var f project.TicketFilter

	if v := q.Get("status"); v != "" {
		s := ticket.Status(v)
		if !s.Valid() {
			return f, fmt.Errorf("invalid status %q", v)
		}
		f.Status = &s
	}
	if v := q.Get("priority"); v != "" {
		p := ticket.Priority(v)
		if !p.Valid() {
			return f, fmt.Errorf("invalid priority %q", v)
		}
		f.Priority = &p
	}
	if v := q.Get("type"); v != "" {
		t := ticket.Type(v)
		if !t.Valid() {
			return f, fmt.Errorf("invalid type %q", v)
		}
		f.Type = &t
	}
	if v := q.Get("tag"); v != "" {
		f.Tag = &v
	}
	if v := q.Get("parent"); v != "" {
		f.Parent = &v
	}
	return f, nil
}
