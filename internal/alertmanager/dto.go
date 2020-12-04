package alertmanager

import "time"

const alertNameLabel = "alertname"

type GetAlertsResponse struct {
	Status string  `json:"status"`
	Alerts []Alert `json:"data"`
}

type Annotations struct {
	Message string `json:"message"`
}

type Status struct {
	State       string        `json:"state"`
	SilencedBy  []interface{} `json:"silencedBy"`
	InhibitedBy []interface{} `json:"inhibitedBy"`
}

type Alert struct {
	Labels       map[string]interface{} `json:"labels,omitempty"`
	Annotations  Annotations            `json:"annotations"`
	StartsAt     time.Time              `json:"startsAt"`
	EndsAt       time.Time              `json:"endsAt"`
	GeneratorURL string                 `json:"generatorURL"`
	Status       Status                 `json:"status"`
	Receivers    []string               `json:"receivers"`
	Fingerprint  string                 `json:"fingerprint"`
}
