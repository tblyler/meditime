IMAGE_REGISTRY = registry.0xdad.com/meditime
IMAGE_TAG = latest

.PHONY: build
build:
	docker build -t "${IMAGE_REGISTRY}:${IMAGE_TAG}" .

.PHONY: deploy
deploy: build
	docker push "${IMAGE_REGISTRY}:${IMAGE_TAG}"
