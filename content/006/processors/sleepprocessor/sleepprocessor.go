package sleepprocessor

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

type Config struct {
	SleepMilliseconds uint64 `mapstructure:"sleep_milliseconds"`
}

type sleepTraceProcessor struct {
	sleepDuration time.Duration
	logger        *zap.Logger
	next          consumer.Traces
}

func (p *sleepTraceProcessor) processTraces(ctx context.Context, td ptrace.Traces) error {
	spanCount := td.SpanCount()
	p.logger.Info("SleepProcessor: Request recebido, processamento assíncrono iniciado",
		zap.Int("span_count", spanCount),
		zap.Duration("sleep_duration", p.sleepDuration),
	)

	// Processar de forma assíncrona - não bloqueia a resposta HTTP
	go func() {
		// Aplicar o delay
		time.Sleep(p.sleepDuration)

		p.logger.Info("SleepProcessor: Delay concluído, enviando para próximo stage")

		// Enviar para o próximo consumer após o delay
		if err := p.next.ConsumeTraces(context.Background(), td); err != nil {
			p.logger.Error("SleepProcessor: Erro ao enviar traces", zap.Error(err))
		}
	}()

	// Retorna imediatamente - cliente recebe resposta rápida
	return nil
}

func (p *sleepTraceProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (p *sleepTraceProcessor) Shutdown(ctx context.Context) error {
	return nil
}

func (p *sleepTraceProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (p *sleepTraceProcessor) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return p.processTraces(ctx, td)
}

func newTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	next consumer.Traces,
) (processor.Traces, error) {
	c := cfg.(*Config)

	sleepDuration := time.Duration(c.SleepMilliseconds) * time.Millisecond
	set.Logger.Info("SleepProcessor: Configurado",
		zap.Uint64("sleep_milliseconds", c.SleepMilliseconds),
		zap.Duration("sleep_duration", sleepDuration),
	)

	return &sleepTraceProcessor{
		sleepDuration: sleepDuration,
		logger:        set.Logger,
		next:          next,
	}, nil
}

func createDefaultConfig() component.Config {
	return &Config{
		SleepMilliseconds: 100,
	}
}

func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType("sleepprocessor"),
		createDefaultConfig,
		processor.WithTraces(newTracesProcessor, component.StabilityLevelDevelopment),
	)
}
