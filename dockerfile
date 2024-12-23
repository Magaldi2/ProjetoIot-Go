# Usar uma imagem base oficial do Go
FROM golang:1.23.4

# Configurar o diretório de trabalho dentro do container
WORKDIR /app

# Copiar arquivos do projeto para o container
COPY . .

RUN chmod +x app
# Baixar e instalar dependências Go
RUN go mod tidy

# Compilar o aplicativo
RUN go build -o app

# Comando para iniciar o aplicativo
CMD ["projeto/app"]
