package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fedebram/showtime/internal/booking"
	notificationserver "github.com/fedebram/showtime/internal/notification/server"
)

// compile time check that httpPublisher satisfies the booking interface.
var _ booking.Publisher = (*httpPublisher)(nil)

// httpPublisher publishes booking's outbox by POSTing it
// to the notification service over HTTP.
type httpPublisher struct {
	url    string // notification service base URL, like http://localhost:8081
	client *http.Client
}

func newHTTPPublisher(url string) *httpPublisher {
	return &httpPublisher{
		url: strings.TrimSuffix(url, "/"),
		// a hanging notification service must not block the relay forever
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Publish sends m to notification. Any answer but 2xx is an error:
// the relay keeps m in the outbox and tries again later.
func (p *httpPublisher) Publish(ctx context.Context, m booking.OutboxMessage) error {
	body, err := json.Marshal(notificationserver.SendRequest{
		ID:        m.ID,
		Kind:      string(m.Kind),
		BookingID: m.BookingID,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url+"/notifications", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		// notification down, unreachable or too slow
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("notification answered %s", resp.Status)
	}
	return nil
}
