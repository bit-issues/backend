FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./main.go


FROM alpine:latest

# Install certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser --home /app appuser

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the binary file from the previous stage
COPY --from=builder --chown=appuser:appuser /out/server /app/server

USER appuser

# Command to run the executable
ENTRYPOINT ["/app/server"]
