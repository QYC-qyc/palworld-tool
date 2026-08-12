BACKEND := paladmin
VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)

.PHONY: all web backend run clean test sav-cli

web:
	cd web && npm install && npm run build

backend:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BACKEND) .

all: web backend

run:
	go run . --config config.yaml

test:
	go test ./...

clean:
	rm -rf $(BACKEND) web/dist dist

sav-cli:
	cd module && pip install -r requirements.txt
