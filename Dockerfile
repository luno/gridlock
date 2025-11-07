FROM golang:1.25-alpine@sha256:d3f0cf7723f3429e3f9ed846243970b20a2de7bae6a5b66fc5914e228d831bbb as server
WORKDIR /
COPY ./ ./
RUN mkdir -p build
RUN apk add git && go build -o build/ github.com/luno/gridlock/server

FROM node:16-alpine@sha256:a1f9d027912b58a7c75be7716c97cfbc6d3099f3a97ed84aa490be9dee20e787 as webapp
COPY web .
RUN npm install && npm run build

FROM alpine:3.22.2@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412
COPY --from=server /build/server ./
COPY --from=webapp /build/ webapp/
RUN apk add --no-cache shadow && \
    useradd -U -u 1000 appuser && \
    chown -R 1000:1000 .
USER 1000
ENTRYPOINT ["./server", "--web_build=./webapp"]
