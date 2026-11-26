# Basic BYOJob worker in Go: build it, run it, and publish it with
# `snapctl byojob publish`.
#
#   make run                      run locally on :8080
#   make run-failing              run locally with FAILURE_MODE=true (10% of items fail)
#   make smoke                    send /health and one /batch to a local worker
#   make test                     run the unit tests
#   make build                    build the image the way snapctl does
#   make docker-run               run that image locally on :8080
#   make publish tag=my-worker-v1 build and publish the worker as my-worker-v1
#
# Publishing uses your default snapctl profile. Pass profile=<name> to use
# another one:
#   make publish tag=my-worker-v1 profile=staging
# Pass SNAPCTL="<command>" to use a snapctl that is not on PATH, and
# SNAPCTL_ARGS for any other snapctl options.
# The job's worker image tag is then `tag`.

SNAPCTL      ?= snapctl
SNAPCTL_ARGS ?=
profile      ?=
# Snapser runs job workers on linux/arm64.
PLATFORM     ?= linux/arm64
tag          ?=
PORT         ?= 8080
IMAGE         = byojob-go:$(or $(tag),local)

.PHONY: help run run-failing smoke test build docker-run publish check-tag

help:
	@sed -n '1,/^$$/p' Makefile | sed 's/^# \{0,1\}//'

run:
	PORT=$(PORT) go run .

# FAILURE_RATE=0.3 make run-failing for a different rate.
run-failing:
	PORT=$(PORT) FAILURE_MODE=true go run .

smoke:
	curl -fsS -o /dev/null -w 'GET  /health %{http_code}\n' localhost:$(PORT)/health
	curl -fsS -w ' POST /batch %{http_code}\n' -X POST localhost:$(PORT)/batch \
		-H 'Content-Type: application/json' \
		-d '{"job_id":"j1","run_id":"r1","batch_id":"b1","attempt":1,"items":["u1","u2","u3"]}'

test:
	go test ./...

# Same platform snapctl builds for, so a build that works here works in publish.
build:
	docker build --platform $(PLATFORM) -t $(IMAGE) -f Dockerfile .

docker-run: build
	docker run --rm --platform $(PLATFORM) -p $(PORT):8080 $(IMAGE)

publish: check-tag
	$(SNAPCTL) byojob publish --tag $(tag) --path . $(if $(profile),--profile $(profile)) $(SNAPCTL_ARGS)

check-tag:
	@test -n "$(tag)" || (echo "tag is required, e.g. make publish tag=my-worker-v1" && exit 1)
