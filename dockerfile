# Do the build in a golang container with the full toolchain
FROM golang:1.22.2 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=1 GOOS=linux go build -o /laundry-notify ./cmd/laundryNotify

# Must use a distro that contains libc for the sqlite3 driver to work
# Copy only the final binary to the distroless container for minimal size
FROM gcr.io/distroless/base-debian12 AS build-release-stage

WORKDIR /app

COPY --from=builder /laundry-notify /app/laundry-notify

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/laundry-notify"]
