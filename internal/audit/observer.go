package audit

import "github.com/paxren/metrics/internal/models"

type Observer interface {
	Notify(event *models.AuditEvent) error
}
