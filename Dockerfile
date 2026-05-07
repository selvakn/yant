# Stage 1: Build all frontend assets (tldraw bundle + vendor libs)
FROM node:24-alpine AS frontend-builder
WORKDIR /build/frontend-build
COPY frontend-build/package.json frontend-build/package-lock.json ./
RUN npm ci --ignore-scripts
COPY frontend-build/ ./
RUN mkdir -p /build/frontend/static/vendor && npm run build

# Stage 2: Build ncnn static library from source.
# ncnn is not packaged for Alpine; we compile and static-link it into the Go binary
# so the runtime image needs no ncnn package at all.
FROM alpine:edge AS ncnn-builder
RUN apk add --no-cache cmake build-base git ninja
RUN git clone --depth 1 --branch 20240410 https://github.com/Tencent/ncnn.git /ncnn
RUN cmake -S /ncnn -B /ncnn/build \
      -GNinja \
      -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
      -DCMAKE_INSTALL_PREFIX=/ncnn/install \
      -DNCNN_SHARED_LIB=OFF \
      -DNCNN_BUILD_TESTS=OFF \
      -DNCNN_BUILD_TOOLS=OFF \
      -DNCNN_BUILD_EXAMPLES=OFF \
      -DNCNN_ENABLE_LTO=ON \
      -DNCNN_USE_OPENMP=OFF \
      -DCMAKE_BUILD_TYPE=Release && \
    cmake --build /ncnn/build -j$(nproc) && \
    cmake --install /ncnn/build

# Stage 3: Build Go binary with CGO against musl + static ncnn.
FROM golang:1.25-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev g++
WORKDIR /build

# Copy ncnn static library and all headers from the cmake install prefix
COPY --from=ncnn-builder /ncnn/install/lib/libncnn.a /usr/local/lib/
COPY --from=ncnn-builder /ncnn/install/include/ /usr/local/include/

COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./

# CGO_LDFLAGS: static-link ncnn + C++ stdlib (no shared lib needed at runtime)
RUN CGO_ENABLED=1 \
    CGO_CFLAGS="-I/usr/local/include" \
    CGO_LDFLAGS="-L/usr/local/lib -lncnn -lstdc++ -lm" \
    go build -tags ncnn -ldflags="-s -w" -o /server ./cmd/server

# Stage 4: Alpine edge runtime.
# ncnn is static-linked into the binary — no ncnn package needed.
# git for note version control; ca-certificates for HTTPS downloads (model files).
# Model files are NOT bundled — downloaded on first start into /data/models (persistent volume).
FROM alpine:edge AS runtime
RUN apk add --no-cache git ca-certificates libstdc++ libgomp && \
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
ENV MODEL_PATH=/data/models/model.ncnn.param
ENV MODEL_BIN_PATH=/data/models/model.ncnn.bin
ENV TOKENIZER_PATH=/data/models/tokenizer.json
ENV SEMANTIC_SEARCH=true

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/app/server"]
CMD ["-addr", ":8080", "-db", "/data/notes.db", "-notes", "/data/notes", "-uploads", "/data/uploads"]
