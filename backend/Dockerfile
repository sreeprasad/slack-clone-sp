FROM golang:1.21 as builder

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -v -o server

FROM alpine:latest  

WORKDIR /root/

COPY --from=builder /app/server .

RUN apk --no-cache add ca-certificates

EXPOSE 8080

CMD ["./server"]

