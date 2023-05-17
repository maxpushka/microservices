FROM golang:1.16-alpine

WORKDIR /app

COPY . .

RUN go build -o service2 .

EXPOSE 8000

CMD ["./service2"]
