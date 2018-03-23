### Multi-stage build
FROM jormungandrk/goa-build as build

COPY . /go/src/github.com/Microkubes/microservice-registration
RUN go install github.com/Microkubes/microservice-registration

### Main
FROM alpine:3.7

RUN apk --no-cache add ca-certificates

COPY --from=build /go/bin/microservice-registration /usr/local/bin/microservice-registration
EXPOSE 8080

ENV API_GATEWAY_URL="http://localhost:8001"

CMD ["/usr/local/bin/microservice-registration"]
