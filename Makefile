OK_COLOR=\033[32;01m
NO_COLOR=\033[0m

lint:
	@echo "$(OK_COLOR)==> Linting$(NO_COLOR)"
	@golangci-lint run

build: lint test
	@echo "$(OK_COLOR)==> Compiling binary$(NO_COLOR)"
	@go build -ldflags '-s -w' -trimpath -o bin/imaginary

test:
	@echo "$(OK_COLOR)==> Testing$(NO_COLOR)"
	@go test

fuzzing15s:
	@echo "$(OK_COLOR)==> Fuzzing: 90 seconds$(NO_COLOR)"
	@./run-fuzz-tests.sh 15s

fuzzing90s:
	@echo "$(OK_COLOR)==> Fuzzing: 90 seconds$(NO_COLOR)"
	@./run-fuzz-tests.sh 90s

fuzzing5m:
	@echo "$(OK_COLOR)==> Fuzzing: 5 minutes$(NO_COLOR)"
	@./run-fuzz-tests.sh 5m

fuzzing1h:
	@echo "$(OK_COLOR)==> Fuzzing: 1 hour$(NO_COLOR)"
	@./run-fuzz-tests.sh 1h

install:
	go get -u .

benchmark: build
	bash benchmark.sh

docker-build:
	@echo "$(OK_COLOR)==> Building Docker image$(NO_COLOR)"
	docker build --no-cache=true --build-arg IMAGINARY_VERSION=$(VERSION) -t sycured/imaginary:$(VERSION) .

docker-push:
	@echo "$(OK_COLOR)==> Pushing Docker image v$(VERSION) $(NO_COLOR)"
	docker push sycured/imaginary:$(VERSION)

docker: docker-build docker-push

.PHONY: lint test benchmark docker-build docker-push docker
