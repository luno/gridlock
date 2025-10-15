FROM golang:1.25-alpine@sha256:aee43c3ccbf24fdffb7295693b6e33b21e01baec1b2a55acc351fde345e9ec34 as server
WORKDIR /
COPY ./ ./
RUN mkdir -p build
RUN apk add git && go build -o build/ github.com/luno/gridlock/server

FROM node:16-alpine@sha256:a1f9d027912b58a7c75be7716c97cfbc6d3099f3a97ed84aa490be9dee20e787 as webapp
COPY web .
RUN npm install && npm run build

FROM alpine:3.22.1@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1
COPY --from=server /build/server ./
COPY --from=webapp /build/ webapp/
ENTRYPOINT ["./server", "--web_build=./webapp"]
