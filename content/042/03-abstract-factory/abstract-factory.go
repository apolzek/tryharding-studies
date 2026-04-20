package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrUnknownProvider = errors.New("provedor desconhecido")
	ErrObjectNotFound  = errors.New("objeto não encontrado")
)

// Storage é o produto abstrato de armazenamento.
type Storage interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Provider() string
}

// Queue é o produto abstrato de mensageria.
type Queue interface {
	Publish(ctx context.Context, topic string, payload []byte) (msgID string, err error)
	Provider() string
}

// CloudFactory é a abstract factory.
type CloudFactory interface {
	NewStorage() Storage
	NewQueue() Queue
	Region() string
}

// ---- AWS ----

type awsFactory struct{ region string }

func (a *awsFactory) NewStorage() Storage { return newMemStorage("aws:s3", a.region) }
func (a *awsFactory) NewQueue() Queue     { return newMemQueue("aws:sqs", a.region) }
func (a *awsFactory) Region() string      { return a.region }

// ---- GCP ----

type gcpFactory struct{ region string }

func (g *gcpFactory) NewStorage() Storage { return newMemStorage("gcp:gcs", g.region) }
func (g *gcpFactory) NewQueue() Queue     { return newMemQueue("gcp:pubsub", g.region) }
func (g *gcpFactory) Region() string      { return g.region }

// Provider é o identificador do provedor.
type Provider string

const (
	ProviderAWS Provider = "aws"
	ProviderGCP Provider = "gcp"
)

func NewCloudFactory(p Provider, region string) (CloudFactory, error) {
	if region == "" {
		return nil, errors.New("região obrigatória")
	}
	switch p {
	case ProviderAWS:
		return &awsFactory{region: region}, nil
	case ProviderGCP:
		return &gcpFactory{region: region}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, p)
	}
}

// Implementações em memória usadas por ambos os provedores apenas para simulação.
type memStorage struct {
	provider, region string
	mu               sync.RWMutex
	objects          map[string][]byte
}

func newMemStorage(provider, region string) *memStorage {
	return &memStorage{provider: provider, region: region, objects: map[string][]byte{}}
}

func (m *memStorage) Provider() string { return m.provider }

func (m *memStorage) Put(ctx context.Context, key string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = append([]byte(nil), data...)
	return nil
}

func (m *memStorage) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.objects[key]
	if !ok {
		return nil, ErrObjectNotFound
	}
	return append([]byte(nil), b...), nil
}

type memQueue struct {
	provider, region string
	mu               sync.Mutex
	seq              int
	published        map[string][][]byte
}

func newMemQueue(provider, region string) *memQueue {
	return &memQueue{provider: provider, region: region, published: map[string][][]byte{}}
}

func (q *memQueue) Provider() string { return q.provider }

func (q *memQueue) Publish(ctx context.Context, topic string, payload []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if topic == "" {
		return "", errors.New("topic vazio")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.seq++
	q.published[topic] = append(q.published[topic], append([]byte(nil), payload...))
	return fmt.Sprintf("%s:%s:%d", q.provider, topic, q.seq), nil
}
