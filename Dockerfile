FROM golang:1.17.1-alpine3.14

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
COPY *.env ./

RUN mkdir /app/images

RUN go build -o /docker-gs-ping

EXPOSE 8080

CMD [ "/docker-gs-ping" ]