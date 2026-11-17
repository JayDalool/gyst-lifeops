FROM golang:1.25-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o lifeops ./cmd/lifeops

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /app/lifeops /app/lifeops
# The binary carries its own pinned time zone database (internal/tzdb, Issue
# #193) and never reads this image's zone files. Stop the build if its
# America/Winnipeg offsets are wrong (read-only; no database, no network).
RUN ["/app/lifeops", "tzinfo", "-check"]
# Outbound provider HTTPS must validate the public certificate chain.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY web /app/web
COPY migrations /app/migrations
COPY seeds /app/seeds

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/lifeops"]
