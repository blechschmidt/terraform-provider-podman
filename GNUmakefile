default: build

build:
	go build -o terraform-provider-podman

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/hashicorp/podman/0.1.0/$$(go env GOOS)_$$(go env GOARCH)
	cp terraform-provider-podman ~/.terraform.d/plugins/registry.terraform.io/hashicorp/podman/0.1.0/$$(go env GOOS)_$$(go env GOARCH)/

test:
	go test ./... -v

testacc:
	TF_ACC=1 go test ./... -v -timeout 120m

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet

clean:
	rm -f terraform-provider-podman

.PHONY: build install test testacc fmt vet lint clean
