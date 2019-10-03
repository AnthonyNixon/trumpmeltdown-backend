FROM golang:1.13.1-alpine as builder

RUN apk add bash ca-certificates git gcc g++ libc-dev
WORKDIR /go/src/github.com/anthonynixon/trumpmeltdown-backend
COPY go.mod .
RUN go mod tidy

FROM builder as binary_builder

COPY . .
RUN CGO_ENABLED=0 GOOS=linus go build -a -o /trumpmeltdown-backend-http .

FROM scratch

COPY --from=binary_builder /trumpmeltdown-backend-http /trumpmeltdown-backend-http
CMD ["./trumpmeltdown-backend-http"]