## Building the Docker image

To build the docker image use `docker build . -t gridlock`

Run the docker image with `docker run -p 80:80 gridlock`

Access the app on `localhost`

## Simulating metrics to the server

Run `go run ./test/cmd`