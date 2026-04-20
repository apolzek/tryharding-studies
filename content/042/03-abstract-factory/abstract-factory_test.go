package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewCloudFactory(t *testing.T) {
	cases := []struct {
		name     string
		prov     Provider
		region   string
		wantErr  error
		wantStor string
		wantQ    string
	}{
		{"aws", ProviderAWS, "us-east-1", nil, "aws:s3", "aws:sqs"},
		{"gcp", ProviderGCP, "europe-west1", nil, "gcp:gcs", "gcp:pubsub"},
		{"desconhecido", "azure", "br", ErrUnknownProvider, "", ""},
		{"região vazia", ProviderAWS, "", errors.New("região obrigatória"), "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := NewCloudFactory(tc.prov, tc.region)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatal("esperado erro")
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if f.NewStorage().Provider() != tc.wantStor {
				t.Fatalf("storage esperado %s, got %s", tc.wantStor, f.NewStorage().Provider())
			}
			if f.NewQueue().Provider() != tc.wantQ {
				t.Fatalf("queue esperado %s, got %s", tc.wantQ, f.NewQueue().Provider())
			}
		})
	}
}

func TestStoragePutGet(t *testing.T) {
	ctx := context.Background()
	f, _ := NewCloudFactory(ProviderAWS, "us-east-1")
	s := f.NewStorage()
	payload := []byte("hello")
	if err := s.Put(ctx, "k1", payload); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, "k1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("esperado %q, got %q", payload, got)
	}
	if _, err := s.Get(ctx, "inexistente"); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("esperado ErrObjectNotFound, got %v", err)
	}
}

func TestStorageCtxCancel(t *testing.T) {
	f, _ := NewCloudFactory(ProviderGCP, "br")
	s := f.NewStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Put(ctx, "k", []byte("x")); err == nil {
		t.Fatal("esperado erro ctx")
	}
	if _, err := s.Get(ctx, "k"); err == nil {
		t.Fatal("esperado erro ctx")
	}
}

func TestQueuePublish(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		topic   string
		wantErr bool
		prefix  string
	}{
		{"aws ok", "t1", false, "aws:sqs:"},
		{"topic vazio", "", true, ""},
	}
	f, _ := NewCloudFactory(ProviderAWS, "us-east-1")
	q := f.NewQueue()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := q.Publish(ctx, tc.topic, []byte("p"))
			if tc.wantErr {
				if err == nil {
					t.Fatal("esperado erro")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(id, tc.prefix) {
				t.Fatalf("prefix esperado %s, got %s", tc.prefix, id)
			}
		})
	}
}

func TestClientUsesFactoryWithoutKnowingConcrete(t *testing.T) {
	// Reforço didático: a função só recebe a interface.
	run := func(f CloudFactory) string {
		return f.NewStorage().Provider() + "|" + f.NewQueue().Provider() + "|" + f.Region()
	}
	aws, _ := NewCloudFactory(ProviderAWS, "us-east-1")
	gcp, _ := NewCloudFactory(ProviderGCP, "sa-east1")
	if run(aws) == run(gcp) {
		t.Fatal("AWS e GCP produziram os mesmos providers")
	}
	if aws.Region() != "us-east-1" || gcp.Region() != "sa-east1" {
		t.Fatal("região incorreta")
	}
}
