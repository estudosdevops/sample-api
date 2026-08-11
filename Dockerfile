FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copia os arquivos de gerenciamento de dependências.
COPY go.mod go.sum* ./

# Baixa as dependências.
RUN apk add --no-cache ca-certificates && go mod download

# Copia o código-fonte.
COPY . .

# Compila a aplicação, criando um executável estático.
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api

# --- Estágio 2: Final ---
# Usa uma imagem "scratch", que é uma imagem vazia, para máxima segurança e tamanho mínimo.
FROM scratch

# Copia apenas o executável compilado do estágio de build.
COPY --from=builder /app/api /sample-api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Expõe a porta que nosso servidor usa.
EXPOSE 8080

# Define o comando para executar a aplicação quando o contêiner iniciar.
ENTRYPOINT ["/sample-api"]