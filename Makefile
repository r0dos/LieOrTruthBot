BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

export GO111MODULE=on

.PHONY: build
build:
	@echo "-- building binary"
	go build \
		-o ./bin/bot \
		./cmd/bot

.PHONY: run
run:
	@echo "-- run Bot"
	./bin/bot

.PHONY: run_nohup
run_nohup:
	@echo "-- run with nohup"
	nohup ./bin/bot &

.PHONY: build_and_run
build_and_run: build run

.PHONY: docker
docker:
	@echo "-- building docker container"
	docker build -f Dockerfile -t lotbot .

.PHONY: docker_run
docker_run:
	@echo "-- starting docker container"
	docker run --name lotbot --rm \
	-v $(pwd)/data:/persis \
	--env DB_URL=/persis/lot_bot.sqlite \
	-d lotbot

.PHONY: start_docker
start_docker: docker docker_run
