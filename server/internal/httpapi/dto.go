package httpapi

import (
	"localtask/internal/project"
	"localtask/internal/ticket"
)

// createTicketRequest mirrors ticket.Ticket for JSON decoding; only Title
// is required (see project.Project.CreateTicket).
type createTicketRequest struct {
	ID                 string                       `json:"id,omitempty"`
	Title              string                       `json:"title"`
	Description        string                       `json:"description,omitempty"`
	Type               string                       `json:"type,omitempty"`
	Status             string                       `json:"status,omitempty"`
	Priority           string                       `json:"priority,omitempty"`
	Parent             string                       `json:"parent,omitempty"`
	DependsOn          []string                     `json:"depends_on,omitempty"`
	AcceptanceCriteria []ticket.AcceptanceCriterion `json:"acceptance_criteria,omitempty"`
	Constraints        []string                     `json:"constraints,omitempty"`
	Tags               []string                     `json:"tags,omitempty"`
	RelatedDocs        []string                     `json:"related_docs,omitempty"`
	RelatedCode        []string                     `json:"related_code,omitempty"`
}

func (r createTicketRequest) toTicket() ticket.Ticket {
	return ticket.Ticket{
		ID:                 r.ID,
		Title:              r.Title,
		Description:        r.Description,
		Type:               ticket.Type(r.Type),
		Status:             ticket.Status(r.Status),
		Priority:           ticket.Priority(r.Priority),
		Parent:             r.Parent,
		DependsOn:          r.DependsOn,
		AcceptanceCriteria: r.AcceptanceCriteria,
		Constraints:        r.Constraints,
		Tags:               r.Tags,
		RelatedDocs:        r.RelatedDocs,
		RelatedCode:        r.RelatedCode,
	}
}

// updateTicketRequest is a partial update: a field absent from the request
// body leaves the corresponding ticket field untouched (see
// project.TicketPatch).
type updateTicketRequest struct {
	Title              *string                       `json:"title"`
	Description        *string                       `json:"description"`
	Type               *string                       `json:"type"`
	Status             *string                       `json:"status"`
	Priority           *string                       `json:"priority"`
	Parent             *string                       `json:"parent"`
	DependsOn          *[]string                     `json:"depends_on"`
	AcceptanceCriteria *[]ticket.AcceptanceCriterion `json:"acceptance_criteria"`
	Constraints        *[]string                     `json:"constraints"`
	Tags               *[]string                     `json:"tags"`
	RelatedDocs        *[]string                     `json:"related_docs"`
	RelatedCode        *[]string                     `json:"related_code"`
}

func (r updateTicketRequest) toPatch() project.TicketPatch {
	patch := project.TicketPatch{
		Title:              r.Title,
		Description:        r.Description,
		Parent:             r.Parent,
		DependsOn:          r.DependsOn,
		AcceptanceCriteria: r.AcceptanceCriteria,
		Constraints:        r.Constraints,
		Tags:               r.Tags,
		RelatedDocs:        r.RelatedDocs,
		RelatedCode:        r.RelatedCode,
	}
	if r.Type != nil {
		t := ticket.Type(*r.Type)
		patch.Type = &t
	}
	if r.Status != nil {
		s := ticket.Status(*r.Status)
		patch.Status = &s
	}
	if r.Priority != nil {
		p := ticket.Priority(*r.Priority)
		patch.Priority = &p
	}
	return patch
}
