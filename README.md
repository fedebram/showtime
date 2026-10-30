# Showtime

Showtime is a small booking system for a movie theater. You find a show, pick
your seats and confirm the booking. Until you confirm, your seats are held for
10 minutes and you can still change or cancel the booking.

## How it works

1. **Find a show.** `GET /shows` lists every show: the movie, the room and
   when it starts.
2. **See the seat map.** `GET /shows/{id}/seats` shows every seat of the room,
   free or held.
3. **Create a booking.** `POST /bookings` with one reservation per show: the
   exact seats you want, or just how many. The booking is pending and holds
   its seats for 10 minutes. The answer contains a token: keep it, it's the
   only way back to the booking.
4. **Adjust it.** While pending, add a show or remove one.
5. **Confirm or cancel.** Confirming makes the seats yours and sends a
   confirmation. Cancelling gives them back. If you do neither, the booking
   expires after 10 minutes.

Every call after step 3 needs the token, as `Authorization: Bearer <token>`.
The full API, with examples you can try, is at `/docs` once it's running.

## Run with containers

You need Docker with Compose, or Podman with podman-compose.

Clone the repository:

```bash
git clone https://github.com/fedebram/showtime.git
cd showtime
```

or download just the compose file, it runs the published image:

```bash
curl -O https://raw.githubusercontent.com/fedebram/showtime/main/compose.yaml
```

Then start it:

```bash
docker compose up -d      # or: podman compose up -d
```

Open http://localhost:8000/docs and try the API from there.

`docker compose down` stops it and keeps the data. Add `-v` to delete the
data too.

## Run with Go

You need Go 1.27 or newer ([install Go](https://go.dev/doc/install)). Clone the
repository, then start each service in its own terminal:

```bash
go run ./cmd/showtime serve notification --db notification.db
go run ./cmd/showtime serve booking --db booking.db
go run ./cmd/showtime serve gateway
```

Open http://localhost:8000/docs.

Without `--db`, a service keeps everything in memory and loses it when it
stops. `go run ./cmd/showtime --help` lists every command and flag.
