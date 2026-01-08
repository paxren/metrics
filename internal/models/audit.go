package models

import "time"

type AuditEvent struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

func NewAuditEvent(metrics []string, ipAddress string) *AuditEvent {
	return &AuditEvent{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ipAddress,
	}
}
