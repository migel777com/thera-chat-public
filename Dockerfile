FROM golang:1.21.1

WORKDIR /app

ADD . /app

RUN go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /thera-chat

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/engine/reference/builder/#expose
EXPOSE 8080

# Run
CMD ["/thera-chat"]
	
