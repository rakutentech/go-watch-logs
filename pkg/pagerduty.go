package pkg

import (
	"context"
	"net/http"

	"github.com/PagerDuty/go-pdagent/pkg/eventsapi"
)


type PagerDuty struct {
}

func NewPagerDuty() *PagerDuty {
	return &PagerDuty{}
}

// SendWithOptions sends an event to PagerDuty with additional options
func (pd *PagerDuty) Send(summary string, details map[string]any, routingKey string, severity string, dedupKey string, httpClient *http.Client) (string, error) {
	event := &eventsapi.EventV2{
		RoutingKey:  routingKey,
		EventAction: "trigger",
		DedupKey:    dedupKey,
		Payload: eventsapi.PayloadV2{
			Summary:       summary,
			Source:        summary,
			Severity:      severity,
			CustomDetails: details,
		},
	}

	resp, err := eventsapi.EnqueueV2(context.Background(), httpClient, event)
	if err != nil {
		return "", err
	}

	return resp.Status, nil
}
