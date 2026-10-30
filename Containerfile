FROM docker.io/library/golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /showtime ./cmd/showtime

# multi stage build! smaller final image

# distroless nonroot: runs as the nonroot user.
# home /home/nonroot is where the services keep their SQLite files
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /showtime /showtime
ENTRYPOINT ["/showtime"]
