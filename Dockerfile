## Build the frontend
FROM node:22-alpine AS web-build
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

## Build the Go binaries
FROM golang:1.25-alpine AS server-build
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 go build -o /out/taskboard-server ./cmd/server
RUN CGO_ENABLED=0 go build -o /out/taskboard-mcp ./cmd/mcp

## Runtime image
FROM alpine:3.20
# git is optional (see internal/git) but needed for the git status/branch/
# commits info shown on the dashboard and returned by get_project_context.
RUN apk add --no-cache git
WORKDIR /app
COPY --from=server-build /out/taskboard-server /out/taskboard-mcp ./
COPY --from=web-build /src/web/dist ./dist

EXPOSE 8080

# Binds 0.0.0.0 *inside* the container's own network namespace — this is
# safe only because the compose.yaml port mapping restricts the host side
# to 127.0.0.1, which is the actual localhost-only enforcement point (see
# the -host flag's help text in cmd/server/main.go for why).
ENTRYPOINT ["./taskboard-server"]
CMD ["-config", "/projects/taskboard.config.yaml", "-static", "/app/dist", "-host", "0.0.0.0", "-port", "8080"]
