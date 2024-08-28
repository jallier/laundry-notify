FROM node:22 AS tailwind-builder

WORKDIR /app

COPY . ./

RUN npm install -g tailwindcss
RUN npx tailwindcss -i ./assets/app.css -o ./internal/http/static/app.css --minify

# Do the build in a golang container with the full toolchain
FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY --from=tailwind-builder /app ./

RUN CGO_ENABLED=1 GOOS=linux go build -o /laundry-notify ./cmd/laundryNotify

# Must use a distro that contains libc for the sqlite3 driver to work
# Copy only the final binary to the distroless container for minimal size
FROM gcr.io/distroless/base-debian12 AS build-release-stage

WORKDIR /app

COPY --from=builder /laundry-notify /app/laundry-notify

EXPOSE 8080

# This causes permissions errors with the db and cbf troubleshooting for just me; disabling for now
# USER nonroot:nonroot

ENTRYPOINT ["/app/laundry-notify"]
