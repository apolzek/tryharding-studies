package alertprocessor

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

// AlertRule define uma regra de alerta
type AlertRule struct {
	Name           string `mapstructure:"name"`            // Nome da regra para identificação
	AttributeKey   string `mapstructure:"attribute_key"`   // Key do atributo (opcional)
	AttributeValue string `mapstructure:"attribute_value"` // Value do atributo (opcional)
	ServiceName    string `mapstructure:"service_name"`    // Service name específico (opcional)
}

// Config define a configuração do alertprocessor
type Config struct {
	WebhookURL        string            `mapstructure:"webhook_url"`
	AlertRules        []AlertRule       `mapstructure:"alert_rules"`
	WebhookTimeout    time.Duration     `mapstructure:"webhook_timeout"`
	WebhookHeaders    map[string]string `mapstructure:"webhook_headers"`
	EnabledForTraces  bool              `mapstructure:"enabled_for_traces"`
	EnabledForMetrics bool              `mapstructure:"enabled_for_metrics"`
}

// AlertPayload define o payload que será enviado para o webhook
type AlertPayload struct {
	AlertType      string                 `json:"alert_type"` // "trace" ou "metric"
	RuleName       string                 `json:"rule_name"`  // Nome da regra que disparou
	Timestamp      time.Time              `json:"timestamp"`
	TriggerKey     string                 `json:"trigger_key"`   // Key do atributo que disparou
	TriggerValue   string                 `json:"trigger_value"` // Value do atributo que disparou
	ServiceName    string                 `json:"service_name"`
	ResourceAttrs  map[string]interface{} `json:"resource_attrs"`
	SpanName       string                 `json:"span_name,omitempty"`        // Só para traces
	MetricName     string                 `json:"metric_name,omitempty"`      // Só para métricas
	TraceID        string                 `json:"trace_id,omitempty"`         // Só para traces
	SpanID         string                 `json:"span_id,omitempty"`          // Só para traces
	FullTraceData  interface{}            `json:"full_trace_data,omitempty"`  // Trace completo em JSON
	FullMetricData interface{}            `json:"full_metric_data,omitempty"` // Métrica completa em JSON
}

type alertProcessor struct {
	config      *Config
	logger      *zap.Logger
	httpClient  *http.Client
	nextTraces  consumer.Traces
	nextMetrics consumer.Metrics
}

// Implementa consumer.Traces
func (p *alertProcessor) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	if p.config.EnabledForTraces {
		p.processTraces(ctx, td)
	}

	// Sempre passa para o próximo consumer
	if p.nextTraces != nil {
		return p.nextTraces.ConsumeTraces(ctx, td)
	}
	return nil
}

// Implementa consumer.Metrics
func (p *alertProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	if p.config.EnabledForMetrics {
		p.processMetrics(ctx, md)
	}

	// Sempre passa para o próximo consumer
	if p.nextMetrics != nil {
		return p.nextMetrics.ConsumeMetrics(ctx, md)
	}
	return nil
}

func (p *alertProcessor) processTraces(ctx context.Context, td ptrace.Traces) {
	// Converter trace para JSON uma vez
	traceJSON := p.traceToJSON(td)

	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rs := td.ResourceSpans().At(i)
		serviceName := p.getServiceName(rs.Resource().Attributes())
		resourceAttrs := p.attributesToMap(rs.Resource().Attributes())

		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			ss := rs.ScopeSpans().At(j)

			for k := 0; k < ss.Spans().Len(); k++ {
				span := ss.Spans().At(k)

				basePayload := AlertPayload{
					AlertType:     "trace",
					Timestamp:     time.Now(),
					ServiceName:   serviceName,
					ResourceAttrs: resourceAttrs,
					SpanName:      span.Name(),
					TraceID:       span.TraceID().String(),
					SpanID:        span.SpanID().String(),
					FullTraceData: traceJSON,
				}

				// Verifica regras contra atributos do span
				p.checkRulesAgainstAttributes(span.Attributes(), serviceName, basePayload)

				// Verifica regras contra atributos do resource
				p.checkRulesAgainstAttributes(rs.Resource().Attributes(), serviceName, basePayload)
			}
		}
	}
}

func (p *alertProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) {
	// Converter métrica para JSON uma vez
	metricJSON := p.metricToJSON(md)

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		serviceName := p.getServiceName(rm.Resource().Attributes())
		resourceAttrs := p.attributesToMap(rm.Resource().Attributes())

		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)

			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)

				basePayload := AlertPayload{
					AlertType:      "metric",
					Timestamp:      time.Now(),
					ServiceName:    serviceName,
					ResourceAttrs:  resourceAttrs,
					MetricName:     metric.Name(),
					FullMetricData: metricJSON,
				}

				// Verifica regras contra atributos do resource
				p.checkRulesAgainstAttributes(rm.Resource().Attributes(), serviceName, basePayload)
			}
		}
	}
}

func (p *alertProcessor) checkRulesAgainstAttributes(attrs pcommon.Map, serviceName string, basePayload AlertPayload) {
	for _, rule := range p.config.AlertRules {
		if p.ruleMatches(rule, attrs, serviceName) {
			payload := basePayload
			payload.RuleName = rule.Name

			// Identifica qual atributo disparou a regra
			if rule.AttributeKey != "" {
				if val, exists := attrs.Get(rule.AttributeKey); exists {
					payload.TriggerKey = rule.AttributeKey
					payload.TriggerValue = val.AsString()
				}
			} else if rule.ServiceName != "" {
				payload.TriggerKey = "service.name"
				payload.TriggerValue = serviceName
			}

			p.logger.Info("AlertProcessor: Regra de alerta disparada",
				zap.String("rule_name", rule.Name),
				zap.String("service_name", serviceName),
				zap.String("trigger_key", payload.TriggerKey),
				zap.String("trigger_value", payload.TriggerValue),
				zap.String("type", basePayload.AlertType),
			)

			// Enviar webhook de forma assíncrona
			go p.sendWebhook(payload)
		}
	}
}

func (p *alertProcessor) ruleMatches(rule AlertRule, attrs pcommon.Map, serviceName string) bool {
	// Verifica service.name se especificado na regra
	if rule.ServiceName != "" && rule.ServiceName != serviceName {
		return false
	}

	// Se não tem AttributeKey especificado, só verifica service.name
	if rule.AttributeKey == "" {
		return rule.ServiceName != "" && rule.ServiceName == serviceName
	}

	// Verifica se o atributo existe
	val, exists := attrs.Get(rule.AttributeKey)
	if !exists {
		return false
	}

	// Se tem AttributeValue especificado, verifica se coincide
	if rule.AttributeValue != "" {
		return val.AsString() == rule.AttributeValue
	}

	// Se chegou aqui, a regra coincide (existe o AttributeKey)
	return true
}

func (p *alertProcessor) traceToJSON(td ptrace.Traces) interface{} {
	// Converte trace para um mapa estruturado
	result := make(map[string]interface{})
	resourceSpans := make([]interface{}, 0)

	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rs := td.ResourceSpans().At(i)
		resourceSpan := map[string]interface{}{
			"resource": map[string]interface{}{
				"attributes": p.attributesToMap(rs.Resource().Attributes()),
			},
			"scopeSpans": make([]interface{}, 0),
		}

		scopeSpans := make([]interface{}, 0)
		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			ss := rs.ScopeSpans().At(j)
			spans := make([]interface{}, 0)

			for k := 0; k < ss.Spans().Len(); k++ {
				span := ss.Spans().At(k)
				spanData := map[string]interface{}{
					"traceId":           span.TraceID().String(),
					"spanId":            span.SpanID().String(),
					"name":              span.Name(),
					"kind":              span.Kind().String(),
					"startTimeUnixNano": span.StartTimestamp().String(),
					"endTimeUnixNano":   span.EndTimestamp().String(),
					"attributes":        p.attributesToMap(span.Attributes()),
				}
				spans = append(spans, spanData)
			}

			scopeSpan := map[string]interface{}{
				"scope": map[string]interface{}{
					"name": ss.Scope().Name(),
				},
				"spans": spans,
			}
			scopeSpans = append(scopeSpans, scopeSpan)
		}

		resourceSpan["scopeSpans"] = scopeSpans
		resourceSpans = append(resourceSpans, resourceSpan)
	}

	result["resourceSpans"] = resourceSpans
	return result
}

func (p *alertProcessor) metricToJSON(md pmetric.Metrics) interface{} {
	// Converte métrica para um mapa estruturado
	result := make(map[string]interface{})
	resourceMetrics := make([]interface{}, 0)

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		resourceMetric := map[string]interface{}{
			"resource": map[string]interface{}{
				"attributes": p.attributesToMap(rm.Resource().Attributes()),
			},
			"scopeMetrics": make([]interface{}, 0),
		}

		scopeMetrics := make([]interface{}, 0)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			metrics := make([]interface{}, 0)

			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				metricData := map[string]interface{}{
					"name":        metric.Name(),
					"description": metric.Description(),
					"unit":        metric.Unit(),
					"type":        metric.Type().String(),
				}
				metrics = append(metrics, metricData)
			}

			scopeMetric := map[string]interface{}{
				"scope": map[string]interface{}{
					"name": sm.Scope().Name(),
				},
				"metrics": metrics,
			}
			scopeMetrics = append(scopeMetrics, scopeMetric)
		}

		resourceMetric["scopeMetrics"] = scopeMetrics
		resourceMetrics = append(resourceMetrics, resourceMetric)
	}

	result["resourceMetrics"] = resourceMetrics
	return result
}

func (p *alertProcessor) sendWebhook(payload AlertPayload) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		p.logger.Error("AlertProcessor: Erro ao serializar payload", zap.Error(err))
		return
	}

	req, err := http.NewRequest("POST", p.config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		p.logger.Error("AlertProcessor: Erro ao criar request", zap.Error(err))
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Adicionar headers customizados
	for key, value := range p.config.WebhookHeaders {
		req.Header.Set(key, value)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Error("AlertProcessor: Erro ao enviar webhook",
			zap.Error(err),
			zap.String("webhook_url", p.config.WebhookURL),
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		p.logger.Info("AlertProcessor: Webhook enviado com sucesso",
			zap.Int("status_code", resp.StatusCode),
			zap.String("rule_name", payload.RuleName),
			zap.String("trigger_key", payload.TriggerKey),
			zap.String("trigger_value", payload.TriggerValue),
		)
	} else {
		p.logger.Warn("AlertProcessor: Webhook retornou erro",
			zap.Int("status_code", resp.StatusCode),
			zap.String("webhook_url", p.config.WebhookURL),
		)
	}
}

func (p *alertProcessor) getServiceName(attrs pcommon.Map) string {
	if val, exists := attrs.Get("service.name"); exists {
		return val.AsString()
	}
	return "unknown"
}

func (p *alertProcessor) attributesToMap(attrs pcommon.Map) map[string]interface{} {
	result := make(map[string]interface{})
	attrs.Range(func(k string, v pcommon.Value) bool {
		result[k] = v.AsString()
		return true
	})
	return result
}

// Implementa as interfaces necessárias
func (p *alertProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (p *alertProcessor) Shutdown(ctx context.Context) error {
	return nil
}

func (p *alertProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Factory functions
func newTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	next consumer.Traces,
) (processor.Traces, error) {
	c := cfg.(*Config)

	processor := &alertProcessor{
		config:     c,
		logger:     set.Logger,
		nextTraces: next,
		httpClient: &http.Client{
			Timeout: c.WebhookTimeout,
		},
	}

	set.Logger.Info("AlertProcessor: Configurado para traces",
		zap.String("webhook_url", c.WebhookURL),
		zap.Int("rules_count", len(c.AlertRules)),
		zap.Duration("webhook_timeout", c.WebhookTimeout),
	)

	return processor, nil
}

func newMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	next consumer.Metrics,
) (processor.Metrics, error) {
	c := cfg.(*Config)

	processor := &alertProcessor{
		config:      c,
		logger:      set.Logger,
		nextMetrics: next,
		httpClient: &http.Client{
			Timeout: c.WebhookTimeout,
		},
	}

	set.Logger.Info("AlertProcessor: Configurado para métricas",
		zap.String("webhook_url", c.WebhookURL),
		zap.Int("rules_count", len(c.AlertRules)),
		zap.Duration("webhook_timeout", c.WebhookTimeout),
	)

	return processor, nil
}

func createDefaultConfig() component.Config {
	return &Config{
		WebhookURL:        "",
		AlertRules:        []AlertRule{},
		WebhookTimeout:    30 * time.Second,
		WebhookHeaders:    make(map[string]string),
		EnabledForTraces:  true,
		EnabledForMetrics: true,
	}
}

func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType("alertprocessor"),
		createDefaultConfig,
		processor.WithTraces(newTracesProcessor, component.StabilityLevelDevelopment),
		processor.WithMetrics(newMetricsProcessor, component.StabilityLevelDevelopment),
	)
}
