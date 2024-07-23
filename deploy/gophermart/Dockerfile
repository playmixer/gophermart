FROM golang:1.22 as build

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/gophermart/ ./cmd/gophermart/
COPY docs ./docs
COPY internal ./internal

# Build
WORKDIR /app/cmd/gophermart
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/cmd/gophermart

FROM ubuntu:latest

WORKDIR /app

COPY --from=build /app/cmd/gophermart/gophermart /gophermart

EXPOSE 8080

# Run
CMD ["/gophermart"]