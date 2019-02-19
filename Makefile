VERSION=0
KUBECTL=kubectl

build:
	go build .

dockerize:
	docker build -t k8s-clustering:local .

push-to-registry: dockerize
	docker login
	docker tag k8s-clustering:local su225/k8s-clustering:$(VERSION)
	docker push su225/k8s-clustering:$(VERSION)

deploy-to-k8s: k8sdeploy.yaml
	$(KUBECTL) create -f k8sdeploy.yaml

destroy-k8s-deployment:
	$(KUBECTL) delete -f k8sdeploy.yaml

clean:
	rm -rf k8s-clustering