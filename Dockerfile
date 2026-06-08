FROM node:22-alpine AS admin-builder

WORKDIR /src/admin

COPY admin/package*.json ./
RUN npm ci

COPY admin ./
RUN npm run build

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# For CGO (needed by github.com/chai2010/webp for WebP conversion) on Alpine final image:
# Use alpine builder + musl-dev so the binary is musl-linked and runs in alpine.
# Install build tools + libwebp-dev (provides headers for the webp package).
RUN apk add --no-cache gcc musl-dev libwebp-dev pkgconfig

COPY . .

# CGO_ENABLED=1 required for webp package.
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
