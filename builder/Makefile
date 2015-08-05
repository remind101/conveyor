.PHONY: test build data

IMAGE=remind101/conveyor-builder
DATA_IMAGE=conveyor-builder-data
EMAIL=your_email@example.com

test: build data
	docker run --privileged=true \
		--volumes-from=data \
		-e CACHE=off \
		-e REPOSITORY=ejholmes/captain-test \
		-e BRANCH=master \
		-e SHA=2e4edf57db00d55051c64d1568e2214858a0897d \
		-e DRY=true \
		${IMAGE}

build:
	docker build -t ${IMAGE} .

data: data/.dockercfg data/.ssh
	docker rm data || true
	docker create --name data \
		-v ${PWD}/data/.ssh:/var/run/conveyor/.ssh \
		-v ${PWD}/data/.dockercfg:/var/run/conveyor/.dockercfg \
		ubuntu:14.04

data/.dockercfg:
	cp ${HOME}/.dockercfg data/.dockercfg

data/.ssh:
	ssh-keygen -t rsa -b 4096 -C ${EMAIL} -f data/.ssh/id_rsa -P ""
