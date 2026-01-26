# ----- build stage -----
FROM golang:1.25-alpine AS builder
WORKDIR /src

# (optional but good) certs for fetching modules over https
RUN apk add --no-cache ca-certificates

# Leverage Docker cache by copying go.mod and go.sum first
COPY go.mod go.sum ./
RUN go mod download

# Now copy the rest of the source code
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api

# ----- run stage -----
FROM alpine:3.20
RUN apk --no-cache add ca-certificates

WORKDIR /
COPY --from=builder /out/api /api

EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/api"]
