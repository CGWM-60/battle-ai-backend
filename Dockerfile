FROM node:22-alpine AS admin-builder

WORKDIR /src/admin

COPY admin/package*.json ./
RUN npm ci

COPY admin ./
RUN npm run build

FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# Install libwebp-dev + pkg-config so that github.com/chai2010/webp (used for mandatory WebP avatar conversion)
# can build with CGO. This package uses the C libwebp under the hood.
RUN apt-get update && apt-get install -y --no-install-recommends \
    libwebp-dev \
    pkg-config \
 && rm -rf /var/lib/apt/lists/*

COPY . .

# CGO_ENABLED=1 is required for the webp package (pure CGO=0 build fails with undefined symbols).
# The resulting binary will be dynamically linked against libwebp, which we provide in the final alpine image.
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/go-battle-ia .

FROM alpine:3.22

# ca-certificates for HTTPS, libwebp for runtime (required by the chai2010/webp CGO dependency used for avatar WebP conversion)
RUN apk add --no-cache ca-certificates libwebp

WORKDIR /app

COPY --from=builder /out/go-battle-ia /app/go-battle-ia
COPY --from=admin-builder /src/admin/out /app/admin/out

ENV APP_HOST=0.0.0.0
ENV APP_PORT=8080
ENV GIN_MODE=release

EXPOSE 8080

CMD ["/app/go-battle-ia"]
