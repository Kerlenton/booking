FROM golang

WORKDIR /app

COPY main.go .

RUN go mod init booking-service
RUN go mod tidy
RUN go test .
RUN go build -o booking-service

CMD ["/app/booking-service"]
