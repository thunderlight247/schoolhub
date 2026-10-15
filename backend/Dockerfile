# ---- Build stage ----
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod/go.sum first so dependency downloads are cached across builds
# unless dependencies actually change.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /schoolhub-server ./cmd/server

# ---- Run stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /schoolhub-server .

EXPOSE 8080
ENTRYPOINT ["/app/schoolhub-server"]
