# Abstract Factory

## Problema

Uma aplicação multi-cloud precisa produzir famílias de objetos compatíveis (Storage + Queue) por provedor. Misturar `S3` da AWS com `PubSub` do GCP dentro do mesmo fluxo costuma gerar incompatibilidades operacionais (IAM, regiões, billing). O cliente precisa trocar de provedor sem reescrever lógica.

## Solução

Define uma interface `CloudFactory` que produz `Storage` e `Queue`, e uma implementação concreta por provedor. O cliente recebe `CloudFactory` por injeção e ignora o tipo concreto.

```mermaid
classDiagram
    class CloudFactory {
        <<interface>>
        +NewStorage() Storage
        +NewQueue() Queue
        +Region() string
    }
    class Storage {
        <<interface>>
    }
    class Queue {
        <<interface>>
    }
    class awsFactory
    class gcpFactory
    CloudFactory <|.. awsFactory
    CloudFactory <|.. gcpFactory
    awsFactory ..> Storage : cria aws:s3
    awsFactory ..> Queue : cria aws:sqs
    gcpFactory ..> Storage : cria gcp:gcs
    gcpFactory ..> Queue : cria gcp:pubsub
```

## Cenário de produção

Pipeline de faturamento que grava PDFs em blob storage e publica evento de "invoice criada" numa fila. Em produção roda em AWS; em DR (disaster recovery) roda em GCP. Trocar de provedor é apenas uma flag de configuração.

## Estrutura

- `go.mod`
- `abstract-factory.go` — interfaces, fábricas AWS/GCP e implementações em memória
- `main.go` — mesmo pipeline rodando nos dois provedores
- `abstract-factory_test.go` — testes por família de produtos

## Como rodar

```bash
cd 042/03-abstract-factory && go run .
```

## Como testar

```bash
go test -race -v ./...
```

## Quando usar

- Produtos que precisam ser criados juntos e pertencem a uma mesma família.
- Aplicações multi-cloud ou multi-tenant com stacks inteiras diferentes.
- Código cliente que precisa permanecer agnóstico a implementações.

## Quando NÃO usar

- Quando só existe uma única família (use factory method simples).
- Quando a adição frequente de novos produtos força refatorar todas as fábricas.
- Quando as famílias têm pouco em comum e a abstração fica artificial.

## Trade-offs

Prós: garante coerência entre produtos de uma mesma família, facilita trocar "stack inteira" de uma vez, isola o cliente do fornecedor.
Contras: adiciona camadas de indireção, pode levar a explosão combinatória (N produtos x M famílias) e dificultar evolução se os produtos divergirem muito.
