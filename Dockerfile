# Stage 1: Build all frontend assets (tldraw bundle + vendor libs)
FROM node:24-alpine AS frontend-builder
WORKDIR /build/frontend-build
COPY frontend-build/package.json frontend-build/package-lock.json ./
RUN npm ci --ignore-scripts
COPY frontend-build/ ./
RUN mkdir -p /build/frontend/static/vendor && npm run build

# Stage 2: Build Go binary with CGO against musl (Alpine edge toolchain).
# onnxruntime_go bundles onnxruntime_c_api.h — no system onnxruntime headers needed at build time.
FROM golang:1.25-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 3: Alpine edge runtime.
# onnxruntime 1.24.4 from Alpine edge community is a musl-native build — no gcompat needed.
# onnxruntime-dev provides the unversioned libonnxruntime.so symlink required by dlopen.
# Model files are NOT bundled — downloaded on first start into /data/models (persistent volume).
FROM alpine:edge AS runtime
RUN apk add --no-cache git ca-certificates onnxruntime-dev && \
    adduser -D -u 65532 nonroot && \
    mkdir -p /data/notes /data/uploads /data/models && \
    chown -R nonroot:nonroot /data

COPY --chown=nonroot:nonroot --from=backend-builder /server /app/server

COPY --chown=nonroot:nonroot frontend/templates/ /app/frontend/templates/
COPY --chown=nonroot:nonroot frontend/static/css/ /app/frontend/static/css/
COPY --chown=nonroot:nonroot frontend/static/js/ /app/frontend/static/js/
COPY --chown=nonroot:nonroot --from=frontend-builder /build/frontend/static/vendor/ /app/frontend/static/vendor/
USER nonroot
WORKDIR /app

ENV DB_PATH=/data/notes.db
ENV NOTES_DIR=/data/notes
ENV UPLOADS_DIR=/data/uploads
ENV ONNXRUNTIME_LIB_PATH=/usr/lib/libonnxruntime.so
ENV MODEL_PATH=/data/models/model.onnx
ENV TOKENIZER_PATH=/data/models/tokenizer.json
ENV SEMANTIC_SEARCH=true

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/app/server"]
CMD ["-addr", ":8080", "-db", "/data/notes.db", "-notes", "/data/notes", "-uploads", "/data/uploads"]
