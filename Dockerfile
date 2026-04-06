# Stage 1: Build tldraw frontend bundle
FROM node:24-bookworm-slim AS frontend-builder
WORKDIR /build/frontend-build
COPY frontend-build/package.json frontend-build/package-lock.json ./
RUN npm ci --ignore-scripts
COPY frontend-build/ ./
# vite.config.ts outputs to ../frontend/static/vendor/ relative to frontend-build/
RUN mkdir -p /build/frontend/static/vendor && npm run build

# Stage 2: Build Go binary
FROM golang:1.25-bookworm AS backend-builder
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 3: Minimal runtime image
FROM alpine:3.21 AS runtime

RUN apk add --no-cache ca-certificates && \
    addgroup -S appuser && \
    adduser -S -G appuser -h /data -s /sbin/nologin appuser && \
    mkdir -p /data/notes /data/uploads /app/frontend && \
    chown -R appuser:appuser /data

COPY --from=backend-builder /server /app/server

COPY frontend/templates/ /app/frontend/templates/
COPY frontend/static/ /app/frontend/static/

# Overwrite tldraw bundle with freshly built version from Stage 1
COPY --from=frontend-builder /build/frontend/static/vendor/tldraw-bundle.js /app/frontend/static/vendor/tldraw-bundle.js
COPY --from=frontend-builder /build/frontend/static/vendor/tldraw-bundle.css /app/frontend/static/vendor/tldraw-bundle.css

WORKDIR /app
USER appuser

ENV PORT=8080
ENV DB_PATH=/data/notes.db
ENV NOTES_DIR=/data/notes
ENV UPLOADS_DIR=/data/uploads
ENV GITHUB_CLIENT_ID=""
ENV GITHUB_CLIENT_SECRET=""

EXPOSE 8080

VOLUME ["/data"]

ENTRYPOINT ["/app/server"]
CMD ["-addr", ":8080", "-db", "/data/notes.db", "-notes", "/data/notes", "-uploads", "/data/uploads"]
