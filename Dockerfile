### Multi-stage build
FROM golang:1.8.3-alpine3.6 as build

RUN apk add --no-cache git

COPY . /go/src/github.com/JormungandrK/microservice-registration

WORKDIR /go/src/github.com/JormungandrK/user-microservice

RUN go get -u -v github.com/golang/dep/cmd/dep
RUN dep ensure -v
RUN go install github.com/JormungandrK/microservice-registration


### Main
FROM alpine:3.6

RUN apk --no-cache add ca-certificates

COPY --from=build /go/bin/microservice-registration /usr/local/bin/microservice-registration
COPY emailTemplate.html /emailTemplate.html
EXPOSE 8080

ENV API_GATEWAY_URL="http://localhost:8001"

CMD ["/usr/local/bin/microservice-registration"]
