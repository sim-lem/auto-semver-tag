# Build
FROM golang:1.16-alpine AS build

WORKDIR /usr/app
ADD . /usr/app

RUN apk add --no-cache --update make \
    && rm -f /var/cache/apk/*

RUN go build -o auto-semver-tag

# Runtime
FROM alpine:latest

COPY entrypoint.sh /entrypoint.sh
COPY --from=build /usr/app/auto-semver-tag /auto-semver-tag

ENTRYPOINT ["sh", "/entrypoint.sh"]