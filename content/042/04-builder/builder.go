package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ErrNoBaseURL = errors.New("base url obrigatória")
	ErrNoMethod  = errors.New("método HTTP obrigatório")
)

// RequestBuilder acumula configuração e constrói um *http.Request.
type RequestBuilder struct {
	client   *http.Client
	method   string
	baseURL  string
	path     string
	headers  http.Header
	query    map[string]string
	timeout  time.Duration
	bodyJSON any
	err      error
}

func NewRequest(client *http.Client) *RequestBuilder {
	if client == nil {
		client = http.DefaultClient
	}
	return &RequestBuilder{
		client:  client,
		headers: http.Header{},
		query:   map[string]string{},
	}
}

func (b *RequestBuilder) Method(m string) *RequestBuilder  { b.method = m; return b }
func (b *RequestBuilder) BaseURL(u string) *RequestBuilder { b.baseURL = u; return b }
func (b *RequestBuilder) Path(p string) *RequestBuilder    { b.path = p; return b }

func (b *RequestBuilder) Header(k, v string) *RequestBuilder {
	b.headers.Set(k, v)
	return b
}

func (b *RequestBuilder) BearerAuth(token string) *RequestBuilder {
	b.headers.Set("Authorization", "Bearer "+token)
	return b
}

func (b *RequestBuilder) Query(k, v string) *RequestBuilder {
	b.query[k] = v
	return b
}

func (b *RequestBuilder) Timeout(d time.Duration) *RequestBuilder {
	b.timeout = d
	return b
}

func (b *RequestBuilder) JSON(v any) *RequestBuilder {
	b.bodyJSON = v
	b.headers.Set("Content-Type", "application/json")
	return b
}

// Build valida e monta o *http.Request.
func (b *RequestBuilder) Build(ctx context.Context) (*http.Request, error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.baseURL == "" {
		return nil, ErrNoBaseURL
	}
	if b.method == "" {
		return nil, ErrNoMethod
	}
	var body io.Reader
	if b.bodyJSON != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(b.bodyJSON); err != nil {
			return nil, fmt.Errorf("encode body: %w", err)
		}
		body = buf
	}
	url := b.baseURL + b.path
	if len(b.query) > 0 {
		sep := "?"
		for k, v := range b.query {
			url += fmt.Sprintf("%s%s=%s", sep, k, v)
			sep = "&"
		}
	}
	req, err := http.NewRequestWithContext(ctx, b.method, url, body)
	if err != nil {
		return nil, err
	}
	for k, vs := range b.headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	return req, nil
}

// Do constrói e executa o request, devolvendo o corpo lido.
func (b *RequestBuilder) Do(ctx context.Context) (int, []byte, error) {
	if b.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, b.timeout)
		defer cancel()
	}
	req, err := b.Build(ctx)
	if err != nil {
		return 0, nil, err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}
