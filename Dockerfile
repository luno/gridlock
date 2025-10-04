FROM golang:1.25-alpine@sha256:b6ed3fd0452c0e9bcdef5597f29cc1418f61672e9d3a2f55bf02e7222c014abd as server
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
