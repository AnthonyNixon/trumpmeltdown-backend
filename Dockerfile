FROM golang:1.12 as builder

WORKDIR /go/src/github.com/anthonynixon/trumpmeltdown-backend
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -v -o trumpmeltdown-backend-http

FROM alpine
COPY --from=builder /go/src/github.com/anthonynixon/trumpmeltdown-backend/trumpmeltdown-backend-http /trumpmeltdown-backend-http
RUN chmod +x /trumpmeltdown-backend-http

CMD ["/trumpmeltdown-backend-http"]