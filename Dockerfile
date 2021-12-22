### Multi-stage build
FROM golang:1.17.3-alpine3.15 as build

RUN apk --no-cache add git ca-certificates

COPY . /go/src/github.com/Microkubes/microservice-registration

RUN cd /go/src/github.com/Microkubes/microservice-registration && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install


### Main
FROM alpine:3.15

COPY --from=build /go/src/github.com/Microkubes/microservice-registration/config.json /config.json
COPY --from=build /go/bin/microservice-registration /usr/local/bin/microservice-registration
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 8080

CMD ["/usr/local/bin/microservice-registration"]
