FROM golang:1.23-alpine as builder

WORKDIR /app

COPY ./ . 
RUN go mod download
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -o bookly ./cart-service/cmd/

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/bookly .
COPY --from=builder /app/wait-for-it.sh .
COPY --from=builder /app/migrations migrations  

RUN apk add --no-cache bash
RUN chmod +x wait-for-it.sh bookly
CMD ["./wait-for-it.sh", "db:5432", "--timeout=30", "--", "./bookly", "-debug"]

EXPOSE 8082
