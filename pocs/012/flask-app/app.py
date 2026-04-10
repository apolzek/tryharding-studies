from flask import Flask, jsonify, request
from prometheus_client import make_wsgi_app, Counter, Histogram, Gauge, Summary
from werkzeug.middleware.dispatcher import DispatcherMiddleware
import time
import random

app = Flask(__name__)

app.wsgi_app = DispatcherMiddleware(app.wsgi_app, {
    '/metrics': make_wsgi_app()
})

# Definindo as métricas
REQUEST_COUNT = Counter(
    'app_request_count',
    'Application Request Count',
    ['method', 'endpoint', 'http_status']
)
REQUEST_LATENCY = Histogram(
    'app_request_latency_seconds',
    'Application Request Latency',
    ['method', 'endpoint']
)
GAUGE_METRIC = Gauge(
    'app_gauge_metric',
    'A gauge metric that can go up and down'
)
# Nova métrica do tipo Histogram
REQUEST_SIZE_HISTOGRAM = Histogram(
    'app_request_size_bytes',
    'Histogram of request sizes in bytes',
    ['method', 'endpoint']
)
# Nova métrica do tipo Summary
REQUEST_LATENCY_SUMMARY = Summary(
    'app_request_latency_summary_seconds',
    'Summary of request latencies',
    ['method', 'endpoint']
)

@app.route('/')
def hello():
    start_time = time.time()
    REQUEST_COUNT.labels('GET', '/', 200).inc()
    
    # Adiciona um delay aleatório entre 0.1 e 2 segundos
    delay = random.uniform(0.1, 2.0)
    time.sleep(delay)
    
    response = jsonify(message='Hello, world!')
    latency = time.time() - start_time
    REQUEST_LATENCY.labels('GET', '/').observe(latency)
    REQUEST_LATENCY_SUMMARY.labels('GET', '/').observe(latency)
    return response

@app.route('/increment')
def increment_gauge():
    GAUGE_METRIC.inc()
    REQUEST_SIZE_HISTOGRAM.labels('GET', '/increment').observe(len(request.data))
    return jsonify(message='Gauge incremented')

@app.route('/decrement')
def decrement_gauge():
    GAUGE_METRIC.dec()
    REQUEST_SIZE_HISTOGRAM.labels('GET', '/decrement').observe(len(request.data))
    return jsonify(message='Gauge decremented')

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
