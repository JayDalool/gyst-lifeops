FROM golang:1.25-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o lifeops ./cmd/lifeops

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /app/lifeops /app/lifeops
# Outbound provider HTTPS must validate the public certificate chain.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY web /app/web
COPY migrations /app/migrations
COPY seeds /app/seeds

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/lifeops"]
