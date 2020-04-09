.PHONY: update master release setup update_master update_release build

setup:
	git config --global --add url."git@gitlab.com:".insteadOf "https://gitlab.com/"

update:
	rm -rf vendor/
	go mod vendor
	-GOFLAGS="" go get -u all

build:
	go build ./...
	go mod tidy

update_project:
	GOFLAGS="" go get -u gitlab.com/elixxir/primitives@Dora/UnifiedPolling
	GOFLAGS="" go get -u gitlab.com/elixxir/crypto@Dora/GenericSigning
	GOFLAGS="" go get -u gitlab.com/elixxir/comms@Dora/UnifiedPolling
	GOFLAGS="" go get -u gitlab.com/elixxir/gpumaths@release

update_release:
	GOFLAGS="" go get -u gitlab.com/elixxir/primitives@release
	GOFLAGS="" go get -u gitlab.com/elixxir/crypto@release
	GOFLAGS="" go get -u gitlab.com/elixxir/comms@release
	GOFLAGS="" go get -u gitlab.com/elixxir/gpumaths@release

update_master:
	GOFLAGS="" go get -u gitlab.com/elixxir/primitives@master
	GOFLAGS="" go get -u gitlab.com/elixxir/crypto@master
	GOFLAGS="" go get -u gitlab.com/elixxir/comms@master

master: update_master update build

release: update_release update build

project: update_project update build