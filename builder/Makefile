.PHONY: test

test:
	captain build
	docker run --privileged=true \
		--volumes-from=data \
		-e REPOSITORY=ejholmes/captain-test \
		-e BRANCH=master \
		-e SHA=2e4edf57db00d55051c64d1568e2214858a0897d \
		remind101/conveyor-builder
