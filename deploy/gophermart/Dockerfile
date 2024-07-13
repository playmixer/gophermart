FROM golang:1.22 as build

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/gophermart/main.go ./
COPY docs ./docs
COPY internal ./internal

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/gophermart

FROM ubuntu:latest

WORKDIR /app

COPY --from=build /app/gophermart /gophermart

EXPOSE 8080

# Run
CMD ["/gophermart"]