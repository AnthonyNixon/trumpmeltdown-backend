FROM golang:1.12 as builder

WORKDIR /go/src/github.com/anthonynixon/trumpmeltdown-backend
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -v -o trumpmeltdown-backend-http

FROM alpine
COPY --from=builder /go/src/github.com/anthonynixon/trumpmeltdown-backend/trumpmeltdown-backend-http /trumpmeltdown-backend-http
COPY phrases.json /phrases.json
RUN chmod +x /trumpmeltdown-backend-http

RUN apk update \
        && apk upgrade \
        && apk add --no-cache \
        ca-certificates \
        && update-ca-certificates 2>/dev/null || true

CMD ["/trumpmeltdown-backend-http"]