FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o menderartifactsconsumer .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/menderartifactsconsumer .

CMD ["./menderartifactsconsumer"]
