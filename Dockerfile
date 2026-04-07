# Stage 1: Build tldraw frontend bundle
FROM node:24-bookworm-slim AS frontend-builder
WORKDIR /build/frontend-build
COPY frontend-build/package.json frontend-build/package-lock.json ./
RUN npm ci --ignore-scripts
COPY frontend-build/ ./
# vite.config.ts outputs to ../frontend/static/vendor/ relative to frontend-build/
RUN mkdir -p /build/frontend/static/vendor && npm run build

# Stage 2: Download ONNX Runtime + embedding model
FROM debian:bookworm-slim AS model-downloader
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && rm -rf /var/lib/apt/lists/*
ARG ONNX_VERSION=1.22.0
ARG TARGETARCH
RUN ARCH=$([ "$TARGETARCH" = "arm64" ] && echo "aarch64" || echo "x64") && \
    curl -fsSL "https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-linux-${ARCH}-${ONNX_VERSION}.tgz" \
    | tar xz -C /opt && \
    cp /opt/onnxruntime-linux-*/lib/libonnxruntime.so.${ONNX_VERSION} /usr/local/lib/libonnxruntime.so
RUN mkdir -p /models && \
    curl -fsSL -o /models/model.onnx "https://huggingface.co/optimum/all-MiniLM-L6-v2/resolve/main/model.onnx" && \
    curl -fsSL -o /models/tokenizer.json "https://huggingface.co/optimum/all-MiniLM-L6-v2/resolve/main/tokenizer.json"

# Stage 3: Build Go binary (CGO required for onnxruntime_go)
FROM golang:1.25-bookworm AS backend-builder
COPY --from=model-downloader /usr/local/lib/libonnxruntime.so /usr/local/lib/libonnxruntime.so
RUN ldconfig
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 4: Minimal runtime image
FROM debian:bookworm-slim AS runtime

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd -r appuser && \
    useradd -r -g appuser -d /data -s /sbin/nologin appuser && \
    mkdir -p /data/notes /data/uploads /app/frontend /app/models && \
    chown -R appuser:appuser /data

COPY --from=model-downloader /usr/local/lib/libonnxruntime.so /usr/local/lib/libonnxruntime.so
RUN ldconfig

COPY --from=model-downloader /models/ /app/models/
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
ENV ONNXRUNTIME_LIB_PATH=/usr/local/lib/libonnxruntime.so
ENV MODEL_PATH=/app/models/model.onnx
ENV TOKENIZER_PATH=/app/models/tokenizer.json
ENV SEMANTIC_SEARCH=true

EXPOSE 8080

VOLUME ["/data"]

ENTRYPOINT ["/app/server"]
CMD ["-addr", ":8080", "-db", "/data/notes.db", "-notes", "/data/notes", "-uploads", "/data/uploads"]
