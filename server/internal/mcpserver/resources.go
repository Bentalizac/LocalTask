package mcpserver

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"localtask/internal/document"
)

// Resource URIs follow spec §19:
//
//	project://<project>
//	project://<project>/context
//	project://<project>/ticket/<ticket-id>
//	project://<project>/doc/<path>
const uriScheme = "project://"

// splitResourceURI extracts the project name and remaining path segment
// from a "project://" URI. It does its own parsing rather than relying on
// SDK-level template variable extraction, since the URI scheme is entirely
// ours to define.
func splitResourceURI(uri string) (proj, rest string, ok bool) {
	trimmed, found := strings.CutPrefix(uri, uriScheme)
	if !found {
		return "", "", false
	}
	proj, rest, _ = strings.Cut(trimmed, "/")
	if proj == "" {
		return "", "", false
	}
	return proj, rest, true
}

func jsonResourceResult(uri string, v any) (*mcp.ReadResourceResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: "application/json", Text: string(data)}},
	}, nil
}

func (s *server) registerResources(srv *mcp.Server) {
	srv.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: uriScheme + "{project}",
		Name:        "project",
		Description: "Metadata for a configured project.",
		MIMEType:    "application/json",
	}, s.readProjectResource)

	srv.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: uriScheme + "{project}/context",
		Name:        "project-context",
		Description: "Concise project context: description, active/blocked tickets, documentation.",
		MIMEType:    "application/json",
	}, s.readProjectContextResource)

	srv.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: uriScheme + "{project}/ticket/{ticket}",
		Name:        "ticket",
		Description: "Complete information for one ticket.",
		MIMEType:    "application/json",
	}, s.readTicketResource)

	srv.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: uriScheme + "{project}/doc/{+path}",
		Name:        "doc",
		Description: "Raw Markdown documentation from within a project.",
		MIMEType:    "text/markdown",
	}, s.readDocResource)
}

func (s *server) readProjectResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI
	name, _, ok := splitResourceURI(uri)
	if !ok {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	p, err := s.project(name)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	return jsonResourceResult(uri, toProjectInfo(name, p))
}

func (s *server) readProjectContextResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI
	name, _, ok := splitResourceURI(uri)
	if !ok {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	_, res, err := s.getProjectContext(ctx, nil, projectArgs{Project: name})
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	return jsonResourceResult(uri, res)
}

func (s *server) readTicketResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI
	name, rest, ok := splitResourceURI(uri)
	if !ok {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	ticketID, ok := strings.CutPrefix(rest, "ticket/")
	if !ok || ticketID == "" {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	p, err := s.project(name)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	t, err := p.GetTicket(ticketID)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	return jsonResourceResult(uri, t)
}

func (s *server) readDocResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI
	name, rest, ok := splitResourceURI(uri)
	if !ok {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	docPath, ok := strings.CutPrefix(rest, "doc/")
	if !ok || docPath == "" {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	p, err := s.project(name)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	doc, err := document.Read(p.Root, docPath)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: "text/markdown", Text: doc.Content}},
	}, nil
}
