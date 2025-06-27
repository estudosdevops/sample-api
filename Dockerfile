FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copia os arquivos de gerenciamento de dependências.
COPY go.mod go.sum* ./

# Baixa as dependências.
RUN go mod download

# Copia o código-fonte.
COPY . .

# Compila a aplicação, criando um executável estático.
RUN CGO_ENABLED=0 GOOS=linux go build -o /sample-api .

# --- Estágio 2: Final ---
# Usa uma imagem "scratch", que é uma imagem vazia, para máxima segurança e tamanho mínimo.
FROM scratch

# Copia apenas o executável compilado do estágio de build.
COPY --from=builder /sample-api /sample-api

# Expõe a porta que nosso servidor usa.
EXPOSE 8080

# Define o comando para executar a aplicação quando o contêiner iniciar.
ENTRYPOINT ["/sample-api"]