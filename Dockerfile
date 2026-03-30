FROM golang:alpine

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o ./bin ./cmd

CMD [ "/app/bin" ]
EXPOSE 8080