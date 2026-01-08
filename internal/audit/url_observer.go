package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/paxren/metrics/internal/models"
)

type URLObserver struct {
	url    string
	client *http.Client
}

func NewURLObserver(url string) *URLObserver {
	return &URLObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (u *URLObserver) Notify(event *models.AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	resp, err := u.client.Post(u.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
