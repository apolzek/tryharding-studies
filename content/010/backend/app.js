// app.js
// Importar instrumentação ANTES de qualquer outro módulo
require('./instrumentation');
const { W3CTraceContextPropagator } = require('@opentelemetry/core');
const { propagation, trace, context, SpanStatusCode } = require('@opentelemetry/api');
const http = require('http');

// Configurar propagador globalmente
propagation.setGlobalPropagator(new W3CTraceContextPropagator());

// Criar tracer específico para nossa aplicação
const tracer = trace.getTracer('items-api', '1.0.0');

let items = [];

// ADICIONE esta validação no início da função extractTraceContext:
function extractTraceContext(headers) {
  const carrier = {};

  // Normalizar headers para lowercase
  Object.keys(headers).forEach(key => {
    const normalizedKey = key.toLowerCase();
    carrier[normalizedKey] = headers[key];
  });

  // DEBUG: Mostrar headers recebidos
  console.log('🔍 Headers recebidos para extração:', {
    traceparent: carrier.traceparent,
    tracestate: carrier.tracestate,
    'x-trace-id': carrier['x-trace-id'],
    allHeaders: Object.keys(carrier).filter(k => k.includes('trace') || k.startsWith('x-'))
  });

  // Verificar se traceparent está presente
  if (!carrier.traceparent) {
    console.log('⚠️ PROBLEMA: traceparent não encontrado nos headers');
    return context.active();
  }

  try {
    const extractedContext = propagation.extract(context.active(), carrier);

    // Verificar se a extração funcionou
    const span = trace.getSpan(extractedContext);
    if (span) {
      const spanContext = span.spanContext();
      console.log('✅ Contexto extraído com sucesso:', {
        traceId: spanContext.traceId,
        spanId: spanContext.spanId,
        traceFlags: spanContext.traceFlags
      });
      return extractedContext;
    } else {
      console.log('❌ Falha na extração: span não encontrado');
      return context.active();
    }
  } catch (error) {
    console.error('❌ Erro na extração:', error.message);
    return context.active();
  }
}
// Função para criar spans customizados para operações de negócio
function createBusinessSpan(spanName, operation, extractedContext, attributes = {}) {
  return new Promise((resolve, reject) => {
    // Usar o contexto extraído se disponível
    const activeContext = extractedContext || context.active();

    const span = tracer.startSpan(spanName, {
      attributes: {
        'service.name': 'items-api',
        'service.operation': spanName,
        ...attributes
      }
    }, activeContext); // Passar contexto extraído

    // Executar operação dentro do contexto do span
    context.with(trace.setSpan(activeContext, span), async () => {
      try {
        const startTime = performance.now();
        const result = await operation();

        // Marcar span como sucesso
        span.setStatus({ code: SpanStatusCode.OK });
        span.setAttributes({
          'operation.success': true,
          'operation.duration_ms': Math.round(performance.now() - startTime)
        });

        resolve(result);
      } catch (error) {
        // Marcar span como erro
        span.setStatus({
          code: SpanStatusCode.ERROR,
          message: error.message
        });
        span.setAttributes({
          'operation.success': false,
          'error.name': error.name,
          'error.message': error.message
        });

        // Registrar evento de erro no span
        span.addEvent('error.occurred', {
          'error.type': error.constructor.name,
          'error.stack': error.stack
        });

        reject(error);
      } finally {
        span.end();
      }
    });
  });
}

const server = http.createServer(async (req, res) => {
  const { method, url } = req;

  // Extrair contexto de trace dos headers ANTES de qualquer operação
  const extractedContext = extractTraceContext(req.headers);

  // Log de debug dos headers de trace
  console.log('🔗 Headers de trace recebidos:', {
    traceparent: req.headers.traceparent,
    tracestate: req.headers.tracestate,
    'x-trace-id': req.headers['x-trace-id']
  });

  // Definir headers CORS logo no início
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers',
    'Content-Type, Authorization, X-Requested-With, traceparent, tracestate, x-trace-id, x-span-id, baggage'
  ); res.setHeader('Access-Control-Max-Age', '86400');
  res.setHeader('Content-Type', 'application/json');

  // Executar toda a lógica dentro do contexto extraído
  await context.with(extractedContext, async () => {
    // Adicionar informações de trace na response para debugging
    const currentSpan = trace.getActiveSpan();
    if (currentSpan) {
      const spanContext = currentSpan.spanContext();
      res.setHeader('X-Trace-Id', spanContext.traceId);
      res.setHeader('X-Span-Id', spanContext.spanId);

      console.log('📊 Span ativo encontrado:', {
        traceId: spanContext.traceId,
        spanId: spanContext.spanId
      });
    } else {
      console.log('⚠️ Nenhum span ativo encontrado');
    }

    // Tratar requisições OPTIONS (preflight)
    if (method === 'OPTIONS') {
      res.writeHead(200);
      res.end();
      return;
    }

    let body = [];

    req.on('data', chunk => {
      body.push(chunk);
    }).on('end', async () => {
      body = Buffer.concat(body).toString();

      try {
        if (url === '/api/items' && method === 'GET') {
          await createBusinessSpan('items.list', async () => {
            // Simular uma pequena operação
            await new Promise(resolve => setTimeout(resolve, 10));

            res.writeHead(200);
            res.end(JSON.stringify(items));

            return items;
          }, extractedContext, {
            'items.count': items.length,
            'http.method': 'GET',
            'http.route': '/api/items'
          });

        } else if (url === '/api/items' && method === 'POST') {
          await createBusinessSpan('items.create', async () => {
            const item = JSON.parse(body);

            // Validação básica
            if (!item.name || item.name.trim() === '') {
              throw new Error('Nome do item é obrigatório');
            }

            items.push(item);

            const response = { message: 'Item added', item };
            res.writeHead(201);
            res.end(JSON.stringify(response));

            return response;
          }, extractedContext, {
            'item.name': JSON.parse(body).name || 'unknown',
            'http.method': 'POST',
            'http.route': '/api/items',
            'request.body.size': body.length
          });

        } else if (url.startsWith('/api/items/') && method === 'PUT') {
          const id = parseInt(url.split('/')[3]);

          await createBusinessSpan('items.update', async () => {
            if (id >= 0 && id < items.length) {
              const updatedItem = JSON.parse(body);

              // Validação básica
              if (!updatedItem.name || updatedItem.name.trim() === '') {
                throw new Error('Nome do item é obrigatório');
              }

              items[id] = updatedItem;

              const response = { message: 'Item updated', updatedItem };
              res.writeHead(200);
              res.end(JSON.stringify(response));

              return response;
            } else {
              const error = new Error('Item not found');
              error.statusCode = 404;
              throw error;
            }
          }, extractedContext, {
            'item.id': id,
            'item.name': JSON.parse(body).name || 'unknown',
            'http.method': 'PUT',
            'http.route': '/api/items/:id',
            'request.body.size': body.length
          });

        } else if (url.startsWith('/api/items/') && method === 'DELETE') {
          const id = parseInt(url.split('/')[3]);

          await createBusinessSpan('items.delete', async () => {
            if (id >= 0 && id < items.length) {
              const deletedItem = items[id];
              items.splice(id, 1);

              const response = { message: 'Item deleted', deletedItem };
              res.writeHead(200);
              res.end(JSON.stringify(response));

              return response;
            } else {
              const error = new Error('Item not found');
              error.statusCode = 404;
              throw error;
            }
          }, extractedContext, {
            'item.id': id,
            'http.method': 'DELETE',
            'http.route': '/api/items/:id',
            'items.remaining': items.length - 1
          });

        } else {
          // Rota não encontrada
          const span = tracer.startSpan('route.not_found', {
            attributes: {
              'http.method': method,
              'http.url': url,
              'http.status_code': 404
            }
          }, extractedContext);

          span.setStatus({ code: SpanStatusCode.ERROR, message: 'Route not found' });

          res.writeHead(404);
          res.end(JSON.stringify({ message: 'Route not found' }));

          span.end();
        }
      } catch (error) {
        // Tratamento global de erros
        console.error('❌ Erro no servidor:', error);

        const statusCode = error.statusCode || 500;
        const message = error.message || 'Internal server error';

        // Adicionar informações do erro ao span ativo
        const currentSpan = trace.getActiveSpan();
        if (currentSpan) {
          currentSpan.recordException(error);
          currentSpan.setStatus({
            code: SpanStatusCode.ERROR,
            message: error.message
          });
        }

        res.writeHead(statusCode);
        res.end(JSON.stringify({
          message,
          error: process.env.NODE_ENV === 'development' ? error.stack : undefined
        }));
      }
    });
  });
});

const PORT = process.env.PORT || 3002;

server.listen(PORT, () => {
  console.log(`🚀 Servidor rodando na porta ${PORT}`);
  console.log(`📊 OpenTelemetry ativo - Traces disponíveis`);
  console.log(`🔗 CORS configurado para frontend`);
  console.log(`🔗 Propagação de contexto ativada`);

  // Criar um span de inicialização
  const span = tracer.startSpan('server.startup', {
    attributes: {
      'server.port': PORT,
      'server.environment': process.env.NODE_ENV || 'development'
    }
  });

  span.addEvent('server.started', {
    'startup.timestamp': new Date().toISOString()
  });

  span.end();
});

// Tratamento de shutdown graceful
process.on('SIGTERM', () => {
  console.log('🛑 Recebido SIGTERM, desligando servidor...');
  server.close(() => {
    console.log('✅ Servidor desligado com sucesso');
  });
});

process.on('SIGINT', () => {
  console.log('🛑 Recebido SIGINT, desligando servidor...');
  server.close(() => {
    console.log('✅ Servidor desligado com sucesso');
  });
});
