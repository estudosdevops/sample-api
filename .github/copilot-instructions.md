# Instruções do Projeto: Sample-API Observabilidade

## Stack Tecnológica
- Linguagem: Go 1.25+
- Web Framework: Gin
- Banco de Dados: PostgreSQL (database/sql ou pgx) + Migrations SQL
- Cache: Redis (go-redis)
- Cliente HTTP Externo: ViaCEP (com `net/http` instrumentado via `otelhttp`)
- Observabilidade: OpenTelemetry SDK (OTLP gRPC), Prometheus, Grafana Alloy/Tempo/Loki
- Logging: `slog` nativo em formato JSON estruturado

## Arquitetura & Padrões
- Seguir estritamente **Clean Architecture** (Domain, UseCase, Infrastructure/Repository/Client, Delivery/HTTP).
- Uso obrigatório de **Interfaces** e Injeção de Dependência manual (sem frameworks mágicos de DI).
- O `context.Context` **deve** ser passado em todas as assinaturas de funções (UseCases, Repositories, Clients).
- **Proibido** o uso de variáveis globais.
- Tratamento rigoroso de erros com wrap (`fmt.Errorf`) e adição de Status Error nos Spans do OTel.

## Padrões de Observabilidade (Obrigatório em todo código gerado)
- **Logs (`slog`)**: Devem conter obrigatoriamente: `trace_id`, `span_id`, `request_id`, `user_id`, `method`, `path`, `latency`, `status`, `service.name`, `service.version`, `environment`.
- **Traces**: Toda operação crítica (DB, Redis, HTTP externo ao ViaCEP, Business Logic) deve ter Span manual criado via `otel.Tracer(...)`. Adicionar atributos OTel padrão (`http.method`, `db.table`, `cache.hit`, `external.service`, `cep`, etc.).
- **Métricas**: Expor contadores e histogramas customizados (`api_requests_total`, `external_api_duration`, etc.).

## Regras para Integração Externa (ViaCEP)
- O client HTTP para o ViaCEP deve implementar uma interface no domínio.
- Deve utilizar OpenTelemetry (`otelhttp` ou span manual) para propagar e registrar a chamada externa.
- Deve tratar falhas de rede, timeouts configuráveis e cenários de *fallback* ou erro devidamente logados com `slog`.