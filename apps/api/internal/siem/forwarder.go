package siem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// Forwarder ships audit events to a SIEM/SOAR webhook in real-time.
// Supports JSON (default) and CEF (ArcSight/QRadar compatible) formats.
// Disabled when WebhookURL is empty — zero overhead in unconfigured deployments.
type Forwarder struct {
	webhookURL string
	format     string
	client     *http.Client
	ch         chan db.CreateAuditLogParams
	logger     *zap.Logger
}

// JSONEvent is the structured payload sent in json format.
type JSONEvent struct {
	Version     string         `json:"@version"`
	Type        string         `json:"@type"`
	Timestamp   time.Time      `json:"@timestamp"`
	EventCode   string         `json:"event_code"`
	Severity    string         `json:"severity"`
	ActorID     string         `json:"actor_id,omitempty"`
	IPAddress   string         `json:"ip_address,omitempty"`
	Resource    string         `json:"resource_type,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	SeqNum      int64          `json:"seq_num"`
	EntryHash   string         `json:"entry_hash"`
	Application string         `json:"application"`
}

func New(webhookURL, format string, logger *zap.Logger) *Forwarder {
	f := &Forwarder{
		webhookURL: webhookURL,
		format:     format,
		client:     &http.Client{Timeout: 5 * time.Second},
		ch:         make(chan db.CreateAuditLogParams, 1000),
		logger:     logger,
	}
	if webhookURL != "" {
		go f.worker()
	}
	return f
}

// Forward queues an audit event for SIEM forwarding.
// Non-blocking — drops the event if the queue is full rather than slowing down the main path.
func (f *Forwarder) Forward(p db.CreateAuditLogParams) {
	if f.webhookURL == "" {
		return
	}
	select {
	case f.ch <- p:
	default:
		f.logger.Warn("siem forwarder queue full — event dropped", zap.String("event_code", p.EventCode))
	}
}

func (f *Forwarder) worker() {
	for p := range f.ch {
		if err := f.send(p); err != nil {
			f.logger.Warn("siem forward failed", zap.Error(err), zap.String("event_code", p.EventCode))
		}
	}
}

func (f *Forwarder) send(p db.CreateAuditLogParams) error {
	var body []byte
	var err error

	switch f.format {
	case "cef":
		body = []byte(toCEF(p))
	default:
		event := toJSON(p)
		body, err = json.Marshal(event)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, f.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpsOwl-Source", "audit-forwarder")

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode >= 400 {
		return fmt.Errorf("siem returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func toJSON(p db.CreateAuditLogParams) JSONEvent {
	e := JSONEvent{
		Version:     "1",
		Type:        "audit_event",
		Timestamp:   time.Now().UTC(),
		EventCode:   p.EventCode,
		Severity:    p.Severity,
		Application: "opsowl",
		SeqNum:      p.SeqNum,
		EntryHash:   p.EntryHash,
	}
	if p.ActorID.Valid {
		e.ActorID = p.ActorID.UUID.String()
	}
	if p.IpAddress.Valid {
		e.IPAddress = p.IpAddress.String
	}
	if p.ResourceType.Valid {
		e.Resource = p.ResourceType.String
	}
	if len(p.Metadata) > 0 {
		_ = json.Unmarshal(p.Metadata, &e.Metadata)
	}
	return e
}

// toCEF produces a CEF 0 formatted string compatible with ArcSight and QRadar.
// CEF:Version|Device Vendor|Device Product|Device Version|Signature ID|Name|Severity|Extension
func toCEF(p db.CreateAuditLogParams) string {
	sev := cefSeverity(p.Severity)
	ext := fmt.Sprintf("rt=%d suser=%s src=%s",
		time.Now().Unix(),
		cefSafe(p.ActorID.UUID.String()),
		cefSafe(p.IpAddress.String),
	)
	return fmt.Sprintf("CEF:0|OpsOwl|InvestigationPlatform|1.0|%s|%s|%d|%s\n",
		p.EventCode, p.EventCode, sev, ext)
}

func cefSeverity(s string) int {
	switch s {
	case "critical":
		return 10
	case "error":
		return 8
	case "warn":
		return 5
	default:
		return 3
	}
}

func cefSafe(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
