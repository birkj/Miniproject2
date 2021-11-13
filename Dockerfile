FROM --platform=${BUILDPLATFORM} golang:1.17-alpine AS base
WORKDIR /src


COPY . .
RUN go build server



