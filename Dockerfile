FROM golang:1.24-alpine as server
WORKDIR /
COPY ./ ./
RUN mkdir -p build
RUN apk add git && go build -o build/ github.com/luno/gridlock/server

FROM node:16-alpine as webapp
COPY web .
RUN npm install && npm run build

FROM alpine:latest
COPY --from=server /build/server ./
COPY --from=webapp /build/ webapp/
RUN apk add --no-cache shadow && \
    useradd -U -u 1000 appuser && \
    chown -R 1000:1000 .
USER 1000
ENTRYPOINT ["./server", "--web_build=./webapp"]
