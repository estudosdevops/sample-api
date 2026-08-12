# sample-api

Laboratório em Go para experimentar padrões modernos de observabilidade e operação de APIs em ambientes locais e Kubernetes.

## Visão Geral

O `sample-api` é um sandbox criado para demonstrar e testar uma stack baseada em:

- [OpenTelemetry](https://opentelemetry.io/) para traces e métricas;
- [Grafana Alloy](https://grafana.com/docs/alloy/) como gateway de telemetria;
- Grafana Beyla e eBPF para observabilidade automática de aplicações;
- Grafana Loki para logs;
- Prometheus/Mimir e Grafana para métricas e dashboards;
- Grafana Tempo para traces.

A função de negócio é consultar endereços a partir de um CEP:

```text
GET /address/:cep
```

O fluxo da consulta é:

1. O CEP é sanitizado, removendo hífens, pontos, espaços e qualquer caractere não numérico.
2. O Redis é consultado usando o CEP sanitizado como chave única.
3. Em caso de cache miss, o PostgreSQL é consultado.
4. Se o endereço não estiver no banco, a API consulta o ViaCEP.
5. O resultado do ViaCEP é persistido no PostgreSQL e no Redis.
6. O endereço é retornado ao cliente.

Assim, entradas como `077.434-05`, `077434-05` e `07743405` referenciam o mesmo registro: `07743405`.

## Arquitetura

O projeto segue uma organização em camadas, com injeção manual de dependências:

```text
cmd/api                  Bootstrap da aplicação e servidor HTTP
internal/domain          Entidades e erros de domínio
internal/usecase         Fluxo de consulta de CEP
internal/delivery/http   Rotas e handlers Gin
internal/repository      Contratos de persistência e cache
internal/repository/     Adaptadores PostgreSQL, Redis e memória
internal/clients/        Cliente HTTP do ViaCEP
internal/infra           OTEL, métricas, logs e utilitários de infraestrutura
alloy/config.alloy       Pipelines OTLP do Grafana Alloy
charts/                  Valores Helm para execução no Kubernetes
```

## Arquitetura de Observabilidade

### Push OTLP via gRPC

A aplicação envia traces e métricas OpenTelemetry por **OTLP gRPC** para o Grafana Alloy na porta `4317`:

```text
sample-api --OTLP gRPC:4317--> Grafana Alloy --> Tempo
                                      └-------> Prometheus/Mimir
```

O endpoint padrão é `alloy:4317` e pode ser alterado com:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
```

A API **não expõe `/metrics`** e não utiliza o modelo Pull do Prometheus. As métricas são enviadas em lotes pelo `PeriodicReader` do OpenTelemetry.

### Métricas de negócio

As métricas são criadas com `otel.Meter("sample-api")` e centralizadas em `internal/infra/metrics.go`.

| Métrica | Finalidade | Atributos |
| --- | --- | --- |
| `sample_api_cache_requests_total` | Total de consultas ao Redis | `result="hit"` ou `result="miss"` |
| `sample_api_lookups_by_uf_total` | Consultas de CEP retornadas com sucesso | `uf`, `source` |
| `sample_api_db_operations_total` | Operações no PostgreSQL | `operation="select"` ou `"insert"`, `status="success"` ou `"error"` |

As origens possíveis em `sample_api_lookups_by_uf_total` são `redis`, `postgres` e `viacep`.

### Grafana Beyla e eBPF

No Kubernetes, o Grafana Beyla pode ser executado como agente eBPF para observar o tráfego da aplicação sem alteração no código Go. Ele captura no nível do kernel as métricas RED HTTP:

- **Rate:** taxa de requisições;
- **Errors:** respostas de erro, como HTTP `400` e `500`;
- **Duration:** duração das requisições, incluindo consultas de latência como p95;
- status HTTP `200`, `400`, `500` e outros códigos;
- rastreamento de rede e comunicação entre processos.

As métricas HTTP automáticas ficam sob responsabilidade do Beyla. O código da aplicação mantém apenas as métricas específicas do negócio.

### Traces

Traces são exportados via OTLP gRPC. A aplicação cria spans para operações relevantes, incluindo:

- requisição HTTP;
- lógica de consulta de endereço;
- Redis;
- PostgreSQL;
- chamada externa ao ViaCEP.

### Logs estruturados

O serviço usa `log/slog` com saída JSON. Os logs de inicialização e operação são estruturados e podem ser coletados pelo Alloy/Loki. Quando disponíveis no contexto, os eventos incluem:

- `trace_id`;
- `span_id`;
- `request_id`;
- `service.name`;
- `service.version`;
- `environment`.

Isso permite navegar de um log no Loki diretamente para o trace correspondente no Tempo.

## Resiliência e Kubernetes

### Retry de dependências

Na inicialização, o serviço testa PostgreSQL e Redis com:

- até 3 tentativas;
- timeout de 2 segundos por tentativa;
- intervalo de 2 segundos entre tentativas;
- logs JSON com endereço, tentativa, status e `latency_ms`.

DSNs e endereços são tratados para não expor senhas nos logs.

### Probes de saúde

#### Liveness

```http
GET /healthz
```

Indica apenas que o processo Go está vivo:

```json
{"status":"healthy"}
```

#### Readiness

```http
GET /readyz
```

Executa Pings ativos no PostgreSQL e Redis, cada um com timeout de 2 segundos.

Quando ambos estão disponíveis:

```json
{
  "status": "ok",
  "postgres": "up",
  "redis": "up"
}
```

Quando uma dependência falha, retorna HTTP `503`:

```json
{
  "status": "unhealthy",
  "postgres": "down",
  "redis": "up",
  "error": "mensagem do erro"
}
```

Os valores podem ser usados diretamente em `livenessProbe` e `readinessProbe` do Kubernetes.

## Como Executar Localmente

### Pré-requisitos

- Go `1.25+`;
- Docker e Docker Compose;
- `curl` para testes;
- Grafana Alloy, caso queira receber traces e métricas localmente.

### Configuração

Copie o arquivo de exemplo:

```bash
cp .env.example .env
```

Principais variáveis:

| Variável | Padrão | Descrição |
| --- | --- | --- |
| `POSTGRES_DSN` | `postgres://postgres:postgres@localhost:5432/sampledb?sslmode=disable` | Conexão PostgreSQL |
| `REDIS_ADDR` | `localhost:6379` | Endereço do Redis |
| `HTTP_PORT` | `8080` | Porta HTTP |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `alloy:4317` | Endpoint OTLP gRPC |
| `SERVICE_NAME` | `sample-api` | Nome do serviço |
| `SERVICE_VERSION` | `0.1.0` | Versão do serviço |
| `ENVIRONMENT` | `dev` | Ambiente de execução |

### Subir dependências

```bash
docker compose up -d postgres redis alloy
```

O PostgreSQL cria a tabela `addresses` usando os scripts em `db/init` na primeira inicialização do volume.

### Executar a API

Com o `.env` carregado:

```bash
set -a
source .env
set +a
go run ./cmd/api
```

Ou usando o Makefile:

```bash
make up
make run
```

### Testes

```bash
go test ./...
go build ./cmd/api
```

## Como Testar a API

Consulta de endereço:

```bash
curl http://localhost:8080/address/07743405
```

O mesmo endereço pode ser consultado com formatação; o serviço normaliza o valor antes de acessar Redis e PostgreSQL:

```bash
curl http://localhost:8080/address/07743-405
```

Liveness:

```bash
curl http://localhost:8080/healthz
```

Readiness:

```bash
curl -i http://localhost:8080/readyz
```

## Consultas PromQL

Depois que o Alloy encaminhar as métricas para Prometheus ou Mimir, exemplos de consultas no Grafana incluem:

### Requisições de cache por resultado

```promql
sum by (result) (
  rate(sample_api_cache_requests_total[5m])
)
```

### Taxa de hit do Redis

```promql
sum(rate(sample_api_cache_requests_total{result="hit"}[5m]))
/
sum(rate(sample_api_cache_requests_total[5m]))
```

### Consultas por UF e origem

```promql
sum by (uf, source) (
  increase(sample_api_lookups_by_uf_total[1h])
)
```

### Operações do PostgreSQL por status

```promql
sum by (operation, status) (
  rate(sample_api_db_operations_total[5m])
)
```

As métricas RED HTTP, como RPS, erros e latência p95, devem ser consultadas nas séries exportadas pelo Beyla. Os nomes exatos dependem da configuração e da versão do agente Beyla instalado no cluster.

## Kubernetes

Os valores Helm em `charts/sample-api/values-prod.yaml` configuram:

- imagem e versão da aplicação;
- endpoint OTLP do Alloy dentro do cluster;
- conexão com PostgreSQL e Redis;
- `livenessProbe` em `/healthz`;
- `readinessProbe` em `/readyz`.

Exemplo de verificação após o deploy:

```bash
kubectl get pods
kubectl describe pod <pod-name>
kubectl port-forward svc/sample-api 8080:80
curl http://localhost:8080/readyz
```

## Licença

Este repositório é um laboratório técnico para estudos e validação de práticas de desenvolvimento, observabilidade e operação em Kubernetes.
