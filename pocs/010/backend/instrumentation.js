/*instrumentation.js*/
// Require dependencies
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-proto');
const { Resource } = require('@opentelemetry/resources');
const { SemanticResourceAttributes } = require('@opentelemetry/semantic-conventions');
const { ConsoleSpanExporter } = require('@opentelemetry/sdk-trace-node');
const { BatchSpanProcessor } = require('@opentelemetry/sdk-trace-node');
const {
    getNodeAutoInstrumentations,
} = require('@opentelemetry/auto-instrumentations-node');
const {
    PeriodicExportingMetricReader,
    ConsoleMetricExporter,
} = require('@opentelemetry/sdk-metrics');


const sdk = new NodeSDK({
    serviceName: 'backend',
    version: '0.0.1',
    traceExporter: new OTLPTraceExporter({

        url: "http://localhost:4318/v1/traces",
        // url: "http://localhost:12347/collect",
        headers: {
            'X-Custom-Header': 'backend',
        },

    }),


    metricReader: new PeriodicExportingMetricReader({
        exporter: new ConsoleMetricExporter(),
    }),
    instrumentations: [
        getNodeAutoInstrumentations({
            '@opentelemetry/instrumentation-http': {
                // Configuração para propagação de contexto
                propagateTraceHeaderCorsUrls: [
                    /http:\/\/localhost:.*/,
                    /127\.0\.0\.1:.*/
                ],
                // Capturar headers de requisição e resposta
                requestHook: (span, request) => {
                    span.setAttributes({
                        'http.request.headers.traceparent': request.headers?.traceparent || 'none',
                        'http.request.headers.tracestate': request.headers?.tracestate || 'none'
                    });
                },
                responseHook: (span, response) => {
                    if (response.getHeaders) {
                        const headers = response.getHeaders();
                        span.setAttributes({
                            'http.response.headers.x_trace_id': headers['x-trace-id'] || 'none'
                        });
                    }
                }
            }
        })
    ]
});

sdk.start();
