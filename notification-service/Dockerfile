FROM golang

WORKDIR /app

COPY main.go .

RUN go mod init notification-service
RUN go mod tidy
RUN go test .
RUN go build -o notification-service

CMD ["/app/notification-service"]
