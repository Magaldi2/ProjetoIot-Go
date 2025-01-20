# Escolha a versão do Go >= 1.16
FROM golang:1.23-alpine

# Instale dependências necessárias
RUN apk add --no-cache bash git

# Defina o diretório de trabalho
WORKDIR /app

# Instale o Air
RUN go install github.com/air-verse/air@latest

# Copie os arquivos necessários
COPY go.mod go.sum ./
RUN go mod download


# Copie o restante do projeto para o contêiner
COPY . .

# Exponha a porta
EXPOSE 8080

# Adicione o binário do Go ao PATH
ENV PATH="/go/bin:${PATH}"

# Comando para rodar o Air
CMD ["air"]

