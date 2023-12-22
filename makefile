run:
	go run .

build:
	go build .

docker-build:
	docker build --tag thera-chat .

docker-run:
	docker run -d -p 8080:8080 --name thera-chat thera-chat
