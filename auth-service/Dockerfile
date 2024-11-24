FROM golang

WORKDIR /app

COPY main.go .

RUN go mod init auth-service
RUN go mod tidy
RUN go test .
RUN go build -o auth-service

CMD ["/app/auth-service"]
