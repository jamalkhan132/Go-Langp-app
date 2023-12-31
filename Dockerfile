FROM golang:latest

WORKDIR /app

COPY . .

RUN go mod download

EXPOSE 9090

CMD ["go", "run", "main.go"]

