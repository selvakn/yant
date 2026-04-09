# Stage 1: Build all frontend assets (tldraw bundle + vendor libs)
FROM node:24-bookworm-slim AS frontend-builder
WORKDIR /build/frontend-build
COPY frontend-build/package.json frontend-build/package-lock.json ./
RUN npm ci --ignore-scripts
COPY frontend-build/ ./
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

# Stage 4: Distroless runtime — no shell, no package manager, no OS utilities.
# Includes only glibc, libstdc++, ca-certificates, and tzdata.
FROM busybox:stable AS data-dirs
RUN mkdir -p /data/notes /data/uploads

FROM gcr.io/distroless/cc-debian12:nonroot AS runtime

COPY --from=data-dirs --chown=nonroot:nonroot /data /data
COPY --from=model-downloader /usr/local/lib/libonnxruntime.so /usr/local/lib/libonnxruntime.so
COPY --from=model-downloader /models/ /app/models/
COPY --from=backend-builder /server /app/server

COPY frontend/templates/ /app/frontend/templates/
COPY frontend/static/css/ /app/frontend/static/css/
COPY frontend/static/js/ /app/frontend/static/js/

# All vendor assets (tldraw, htmx, easymde, mermaid) are built in Stage 1
COPY --from=frontend-builder /build/frontend/static/vendor/ /app/frontend/static/vendor/

WORKDIR /app

ENV PORT=8080
ENV DB_PATH=/data/notes.db
ENV NOTES_DIR=/data/notes
ENV UPLOADS_DIR=/data/uploads
ENV ONNXRUNTIME_LIB_PATH=/usr/local/lib/libonnxruntime.so
ENV MODEL_PATH=/app/models/model.onnx
ENV TOKENIZER_PATH=/app/models/tokenizer.json
ENV SEMANTIC_SEARCH=true
ENV LD_LIBRARY_PATH=/usr/local/lib

EXPOSE 8080

VOLUME ["/data"]

ENTRYPOINT ["/app/server"]
CMD ["-addr", ":8080", "-db", "/data/notes.db", "-notes", "/data/notes", "-uploads", "/data/uploads"]
