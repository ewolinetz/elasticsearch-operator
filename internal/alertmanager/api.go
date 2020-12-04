package alertmanager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/ViaQ/logerr/kverrors"
)

type Alerts struct {
	HeapHigh            bool
	LowWatermark        bool
	DiskAvailabilityLow bool
	WriteRejections     bool
}

type API interface {
	Alerts() (*Alerts, error)
}

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient creates a new AlertManager client
func NewClient(url string, httpClient *http.Client, bearerToken string) *Client {
	return &Client{
		baseURL:    url,
		httpClient: httpClient,
		token:      bearerToken,
	}
}

func (c *Client) Alerts() (*Alerts, error) {
	resp, err := c.allAlerts()
	if err != nil {
		return nil, err
	}

	res := &Alerts{}

	for _, alert := range resp.Alerts {
		raw, ok := alert.Labels[alertNameLabel]
		if !ok {
			continue
		}
		alertName, ok := raw.(string)
		if !ok {
			continue
		}
		if alert.Status.State != "active" {
			continue
		}

		switch strings.ToLower(alertName) {
		case "elasticsearchjvmheapusehigh":
			res.HeapHigh = true
		case "elasticsearchnodediskwatermarkreached":
			res.LowWatermark = true
		case "elasticsearchdiskspacerunninglow":
			res.DiskAvailabilityLow = true
		case "elasticsearchwriterequestsrejectionjumps":
			res.WriteRejections = true
		}
	}

	return res, nil
}

func (c *Client) allAlerts() (*GetAlertsResponse, error) {
	uri := path.Join(c.baseURL, "/api/v1/alerts")
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, kverrors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, kverrors.Wrap(err, "failed to GET endpoint")
	}
	defer func() { _ = res.Body.Close() }()
	var alerts *GetAlertsResponse
	if err := json.NewDecoder(res.Body).Decode(&alerts); err != nil {
		return nil, kverrors.Wrap(err, "failed to decode alerts message")
	}

	return alerts, nil
}
