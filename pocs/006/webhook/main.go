package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type AlertPayload struct {
	AlertType      string                 `json:"alert_type"`
	RuleName       string                 `json:"rule_name"`
	Timestamp      time.Time              `json:"timestamp"`
	TriggerKey     string                 `json:"trigger_key"`
	TriggerValue   string                 `json:"trigger_value"`
	ServiceName    string                 `json:"service_name"`
	ResourceAttrs  map[string]interface{} `json:"resource_attrs"`
	SpanName       string                 `json:"span_name,omitempty"`
	MetricName     string                 `json:"metric_name,omitempty"`
	TraceID        string                 `json:"trace_id,omitempty"`
	SpanID         string                 `json:"span_id,omitempty"`
	FullTraceData  interface{}            `json:"full_trace_data,omitempty"`
	FullMetricData interface{}            `json:"full_metric_data,omitempty"`
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Webhook received from %s", r.RemoteAddr)

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}

	// Parse JSON
	var alert AlertPayload
	if err := json.Unmarshal(body, &alert); err != nil {
		log.Printf("Error parsing JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Show alert details
	log.Println("=====================================")
	log.Printf("ALERT: %s", alert.RuleName)
	log.Printf("Type: %s", alert.AlertType)
	log.Printf("Trigger: %s = %s", alert.TriggerKey, alert.TriggerValue)
	log.Printf("Service: %s", alert.ServiceName)

	if alert.AlertType == "trace" {
		log.Printf("Span: %s", alert.SpanName)
		log.Printf("TraceID: %s", alert.TraceID)
		log.Printf("SpanID: %s", alert.SpanID)
	}

	if alert.AlertType == "metric" {
		log.Printf("Metric: %s", alert.MetricName)
	}

	log.Printf("Timestamp: %s", alert.Timestamp.Format("2006-01-02 15:04:05"))

	// Show resource attributes
	if len(alert.ResourceAttrs) > 0 {
		log.Println("Resource Attributes:")
		for k, v := range alert.ResourceAttrs {
			log.Printf("   %s: %v", k, v)
		}
	}

	log.Println("=====================================")

	// Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success", "message": "Alert received"}`)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "healthy", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
}

func main() {
	http.HandleFunc("/hook", webhookHandler)
	http.HandleFunc("/health", healthHandler)

	port := "8080"

	log.Printf("Webhook Server started on http://localhost:%s", port)
	log.Printf("Webhook endpoint: http://localhost:%s/hook", port)
	log.Printf("Health check: http://localhost:%s/health", port)
	log.Println("=====================================")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
