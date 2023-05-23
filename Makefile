.PHONY: build
build: build-client build-node

.PHONY: build-client
build-client:
	docker build -t ui-blockchain-client -f Dockerfile.client . 

.PHONY: build-node
build-node:
	docker build -t ui-blockchain-node -f Dockerfile.node . 

.PHONY: all
all: 
	go run main.go all

.PHONY: client 
client: 
	go run main.go client 

.PHONY: node 
node: 
	go run main.go node 