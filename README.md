## Prerequisites

The Gridlock server needs a redis database in order to run.
This database must be dedicated solely for this application to prevent key clash.

## Running the app locally for development

To run the backend api:
```
go run ./server
```

and to run the web app:
```
cd web && npm install && npm run start
```

## Simulating metrics to the server

Run
```
go run ./test/cmd
```

## Building the Docker image

To build and run the docker image
```
docker build . -t gridlock
docker run -p 80:80 gridlock --redis=redis://host.docker.internal:6379
```
Access the app on `localhost`
