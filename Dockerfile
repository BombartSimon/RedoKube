FROM golang:1.24 AS builder

WORKDIR /workspace

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o redokube ./cmd/

# Use distroless as minimal base image to package the manager binary
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/redokube /redokube

USER 65532:65532

ENTRYPOINT ["/redokube"]