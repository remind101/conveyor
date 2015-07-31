.PHONY: test build data

IMAGE=remind101/conveyor-builder
DATA_IMAGE=conveyor-builder-data
EMAIL=your_email@example.com

test: build data
	docker run --privileged=true \
		--volumes-from=data \
		-e REPOSITORY=ejholmes/captain-test \
		-e BRANCH=master \
		-e SHA=2e4edf57db00d55051c64d1568e2214858a0897d \
		${IMAGE}

build:
	docker build -t ${IMAGE} .

data: data/.dockercfg data/.ssh
	docker build -f Dockerfile.data -t ${DATA_IMAGE} .
	docker rm data
	docker create --name data ${DATA_IMAGE}

data/.dockercfg:
	cp ${HOME}/.dockercfg data/.dockercfg

data/.ssh:
	ssh-keygen -t rsa -b 4096 -C ${EMAIL} -f data/.ssh/id_rsa -P ""
