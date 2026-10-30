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

If you don't have Docker on your machine, or you prefer a virtual machine, you
can use [Lima](https://lima-vm.io/docs/installation/): it runs Docker in a
small Linux VM with its [docker template](https://lima-vm.io/docs/examples/containers/docker/).
Lima forwards the ports, so http://localhost:8000/docs works from your machine.

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

## Architecture

```
                 ┌─────────┐
  client ───────►│ gateway │ :8000
                 └────┬────┘
                      │
                 ┌────▼────┐          ┌──────────────┐
                 │ booking │─────────►│ notification │
                 │  :8080  │   HTTP   │    :8081     │
                 └────┬────┘          └──────┬───────┘
                      │                      │
                 booking.db           notification.db
```

- **gateway** is the only address clients use. It forwards `/shows` and
  `/bookings` to booking and serves the API docs at `/docs`.
- **booking** handles shows, seats and bookings, with its own database.
- **notification** records a notification for every confirmed or cancelled
  booking, with its own database. It only logs it: a real service would send
  an email. Only booking calls it.

## Abstracting the storage

Each service reaches its data through an interface. This means that the
database can be swapped. In fact there are two database implementations for
each service: one in memory, used in testing, and one with SQLite.

This is nice because another database implementation can be added without
changing anything else. Every implementation must comply with the contract
tests: the same tests run against each of them.

For those who know Go: the shape of the calls, with transactions, is inspired
by the famous [BoltDB](https://github.com/boltdb/bolt) by Ben Johnson, a
key-value store in Go where read and write transactions are run with
callbacks. It lives on as [bbolt](https://github.com/etcd-io/bbolt),
maintained by the etcd team: it's the storage engine of etcd, the database
Kubernetes keeps its state in.

The two calls:

```go
View(ctx, func(tx Tx) error) error    // a read transaction
Update(ctx, func(tx Tx) error) error  // a write transaction: an error rolls everything back
```

SQLite was chosen as the main database because it can be embedded in the
program, and it gives a static, self-contained binary that runs anywhere. The
library is [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite): the C
code of SQLite transpiled to pure Go, so no C compiler is needed. For this
project it is a nice solution.

## The outbox

When a booking is confirmed, two things must happen: booking saves the new
status, and notification must hear about it. They are two services with two
databases, so this can't be one transaction. If booking saved and then crashed
before calling notification, the message would be lost. If notification was
down, the same.

The solution is the [transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html).
Booking doesn't call notification while confirming. In the same transaction as
the booking change, it saves a message in an `outbox` table. Both are saved
or neither is, so the message can't be lost.

Then a **relay** runs in the background inside booking. Every second it reads
the messages not sent yet, oldest first, sends each one to notification and
marks it sent. If notification is down, it stops and tries again the next
second. The messages wait in the outbox, in order. The relay sends through a
`Publisher` interface: in the tests it's a fake that records messages and can
pretend to be down, in production it POSTs to notification.

A message can still arrive twice: if booking crashes after sending but before
marking it sent, the relay sends it again. So notification is an
[idempotent consumer](https://microservices.io/patterns/communication-style/idempotent-consumer.html):
it stores the ID of every message it handled and ignores a repeat. Together,
nothing is lost and nothing is sent twice.

Right now notification only logs the message. It can be improved to send a
real email: then the email provider should get the message ID too, so that it
can drop a repeat as well.

To see the outbox in action, and that no notification is lost even when the
notification service is down:

```bash
docker compose stop notification
```

Confirm a booking: it still answers 200, and booking's log shows the relay
retrying. Then:

```bash
docker compose start notification
docker compose logs -f notification
```

Within about half a minute the message shows up in notification's log.
