FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/go-battle-ia .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/go-battle-ia /app/go-battle-ia

ENV APP_HOST=0.0.0.0
ENV APP_PORT=8080
ENV GIN_MODE=release

EXPOSE 8080

CMD ["/app/go-battle-ia"]
