
# Plano de Implementação: Sample-API Observabilidade (Enterprise Ready)

## 📋 Fases de Execução

- [ ] **Fase 1: Fundação, Arquitetura & Domínio**
  - [ ] Criar estrutura de pastas estrita (Clean Architecture: `cmd/`, `internal/domain/`, `internal/usecase/`, `internal/repository/`, `internal/delivery/http/`, `internal/infra/`).
  - [ ] Definir entidades de domínio puras, erros de domínio customizados e mapeamento determinístico para códigos HTTP (notFound, validation, conflict, unauthorized, infra).
  - [ ] Criar interfaces contratuais para Repositórios (Postgres/Redis) e Client HTTP (ViaCEP).
  - [ ] Implementar DTOs e Mappers para separar modelos de transporte (Request/Response) das entidades de domínio.
  - [ ] Definir interface de transação nos repositórios (`BeginTx/ExecTx/Commit/Rollback`) para que UseCases controlem consistência.

- [ ] **Fase 2: Infraestrutura (Postgres, Redis & ViaCEP Client)**
  - [ ] Implementar conexão PostgreSQL com Connection Pool otimizado e instrumentação OTel (ex.: `otelsql` ou `pgx`), garantindo sanitização de PII em statements.
  - [ ] Implementar conexão Redis com Connection Pool, operações de Get/Set e contadores de cache hit/miss (sem alta cardinalidade nas métricas).
  - [ ] Implementar o Client HTTP ViaCEP utilizando `otelhttp` para propagação automática de contexto, timeouts configuráveis, retries com backoff e circuit breaker.
  - [ ] Configurar timeouts coerentes (handler < client) e contexto cancelável para goroutines/background workers.
  - [ ] Implementar mecanismo de locking para migrations (evitar concorrência).

- [ ] **Fase 3: Observabilidade Core, Middlewares & Logging**
  - [ ] Configurar SDK do OpenTelemetry com `BatchSpanProcessor`, exportador OTLP gRPC e Resource Attributes obrigatórios (`service.name`, `service.version`, `deployment.environment`).
  - [ ] Definir política de sampling por ambiente/rota e documentar limites de volume de traces.
  - [ ] Configurar logger estruturado (`slog` em JSON) injetando automaticamente `trace_id`, `span_id`, `request_id`, `user_id` (nos logs, não em labels de métricas), método, path e latência.
  - [ ] Criar middlewares Gin essenciais:
    - Injeção de `request_id` e contexto OTel (W3C Trace Context).
    - Panic recovery e mapeamento centralizado de erros de domínio para HTTP.
    - Middleware de métricas (expor `/metrics` Prometheus), evitando labels de alta cardinalidade.
    - Middleware de rate limiting, CORS e limite de tamanho de payload.
  - [ ] Instrumentar DB (atributos `db.system`, `db.statement` sanitizado), Redis e clientes HTTP; setar status e eventos em spans nos erros.

- [ ] **Fase 4: Casos de Uso & Chaos Engineering**
  - [ ] Implementar o UseCase principal `GET /address/{cep}` seguindo o fluxo completo: Validação -> Redis GET -> Cache Hit/Miss metric -> ViaCEP (otelhttp + retries + circuit breaker) -> Postgres Insert dentro de Tx -> Redis SET -> Response.
  - [ ] Criar endpoints de debug e simulação de falhas (`/debug/slow`, `/delay`, `/error`, `/panic`, `/timeout`) com parâmetros de query string para testes de chaos (`?delay=`, `?error=`, `?cache=`). Proteger/feature-flag estes endpoints (habilitar somente em non-prod).
  - [ ] Criar contract tests / mocks do client ViaCEP (stub/record-replay) para CI.

- [ ] **Fase 5: Health Probes, Wiring, Infra & Deploy**
  - [ ] Implementar endpoints de saúde: `/healthz` (liveness) e `/readyz` (readiness validando conexões reais com Postgres e Redis). Readiness deve ser leve e estável (avoid flapping).
  - [ ] Implementar o ponto de entrada (`cmd/api/main.go`) com injeção de dependência manual, configuração via package `config` (envs, timeouts, OTLP creds) e suporte a **Graceful Shutdown**.
  - [ ] Criar scripts/arquivos SQL de migrations e seeds iniciais; documentar versão das migrations e procedimento de rollback.
  - [ ] Criar `docker-compose.yml` integrando a API, PostgreSQL, Redis e Grafana Alloy (dev/test).
  - [ ] Documentar deploy/secret strategy (k8s Secrets, Vault, or cloud secret manager).

- [ ] **Fase 6: CI, Tests & Quality**
  - [ ] Esboçar pipeline CI: `gofmt`, `go vet`, `golangci-lint`, unit tests, contract tests (ViaCEP stub), integration smoke tests (Postgres+Redis via docker-compose/testcontainers).
  - [ ] Criar testes unitários para UseCases, mocks para Repos/Clients e testes de integração para `GET /address/{cep}`.
  - [ ] Medir cobertura mínima e adicionar checks de quality gate.

## 🛡️ Decisões Técnicas e Guardrails de Observabilidade (obrigatórios)
 - **Resource attributes:** configurar `service.name`, `service.version`, `deployment.environment`, `service.instance.id` quando aplicável.
 - **TracerProvider:** usar `BatchSpanProcessor` + OTLP gRPC exporter com retry/backoff.
 - **Sampling:** política por ambiente (ex.: 100% em dev, probabilístico em staging/prod) e regras por rota para operações caras.
 - **Métricas:** expor `/metrics` Prometheus; métricas mínimas: `api_requests_total`, `api_request_duration_seconds` (histogram), `external_api_duration_seconds` (histogram), `redis_cache_hit_total`, `redis_cache_miss_total`, `db_query_duration_seconds`.
 - **Cardinalidade:** proibido usar `user_id`, `cep`, IPs como labels; usar nos logs e atributos de spans.
 - **Logs:** `slog` JSON com campos padronizados e sem PII; correlacionar logs ↔ traces (trace_id/span_id/request_id).
 - **Sanitização PII:** sanitizar `db.statement` e payloads; políticas claras do que pode ser logado/traced.
 - **Propagação W3C:** `traceparent`/`tracestate` obrigatórios para todas chamadas HTTP.
 - **Error handling & spans:** mapear erros para status OTel (set status & add event on error).
 - **Metrics vs OTel:** usar OTel for traces/metrics pipeline; Prometheus scrape para app metrics quando aplicável.

## 🔒 Segurança & Operacional
 - **Timeouts e retries:** documentar timeouts coerentes; retries com backoff e circuit breaker para chamados externos.
 - **Rate limiting & size limit:** proteger endpoints públicos.
 - **Debug endpoints:** habilitar apenas com feature flag e auth em staging/dev.
 - **Secrets:** não commitar credenciais; usar secrets manager no deploy.
 - **Retention & GDPR:** revisar logs/traces retention e sanitização.

## ✅ Riscos / Pontos de Atenção
 - alta cardinalidade em métricas → custo/instabilidade em Prometheus;
 - vazamento de PII em spans/metrics/logs;
 - timeouts mal configurados podem causar goroutine leaks;
 - instrumentação incompleta reduz utilidade dos traces.

## ▶️ Próximos passos imediatos (priorizados)
 - [ ] Criar Decision Doc OTel: sampling, exporters, resource attrs, metric names/buckets.
 - [ ] Escrever contratos/interfaces e modelos de erro (domain errors → HTTP status).
 - [ ] Scaffold inicial (pastas + interfaces + middleware) e PR mínimo que implemente `GET /address/{cep}` com logs, trace, metrics e testes unitários.
 - [ ] Montar pipeline CI com integration smoke test (docker-compose/testcontainers).
 - [ ] Revisar política de secrets e habilitar feature-flag para debug endpoints.

---

Se quiser, eu:
- gero Issues/Checklist em formato pronto para criar no tracker (GitHub issues/PR template), ou
- forneço o scaffold inicial (arquivo tree + esqueleto de interfaces/middleware) como texto para você colar.

Qual opção prefere?