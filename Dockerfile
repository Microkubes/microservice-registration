### Multi-stage build
FROM golang:1.8.3-alpine3.6 as build

RUN apk --no-cache add git curl openssh

COPY keys/id_rsa /root/.ssh/id_rsa
RUN chmod 700 /root/.ssh/id_rsa && \
    echo -e "Host github.com\n\tStrictHostKeyChecking no\n" >> /root/.ssh/config && \
    git config --global url."ssh://git@github.com:".insteadOf "https://github.com"

RUN go get -u -v github.com/goadesign/goa/... && \
	go get -u -v github.com/afex/hystrix-go/hystrix && \
	go get -u -v gopkg.in/gomail.v2 && \
	go get -u -v github.com/dgrijalva/jwt-go && \
	go get -u -v github.com/satori/go.uuid && \
	go get -u -v github.com/JormungandrK/microservice-tools

COPY . /go/src/github.com/JormungandrK/microservice-registration
RUN go install github.com/JormungandrK/microservice-registration


### Main
FROM alpine:3.6

RUN apk --no-cache add ca-certificates

COPY --from=build /go/bin/microservice-registration /usr/local/bin/microservice-registration
COPY emailTemplate.html /emailTemplate.html
EXPOSE 8080

ENV API_GATEWAY_URL="http://localhost:8001"

CMD ["/usr/local/bin/microservice-registration"]