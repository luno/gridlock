FROM golang:1.18-alpine as server
WORKDIR /
COPY ./ ./
RUN mkdir -p build
RUN apk add git && go build -o build/ github.com/adamhicks/gridlock/server

FROM node:16-alpine as webapp
COPY web .
RUN npm install && npm run build

FROM alpine:latest
COPY --from=server /build/server ./
COPY --from=webapp /build/ webapp/
ENTRYPOINT ["./server", "--web_build=./webapp"]
