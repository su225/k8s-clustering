VERSION=0

build:
	go build .

dockerize:
	docker build -t k8s-clustering:local .

push-to-registry: dockerize
	docker login
	docker tag k8s-clustering:local su225/k8s-clustering:$(VERSION)
	docker push su225/k8s-clustering:$(VERSION)

clean:
	rm -rf k8s-clustering