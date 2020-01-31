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

update_release:
	GOFLAGS="" go get -u gitlab.com/elixxir/primitives@release
	GOFLAGS="" go get -u gitlab.com/elixxir/crypto@release
	GOFLAGS="" go get -u gitlab.com/elixxir/comms@release
	GOFLAGS="" go get -u gitlab.com/elixxir/gpumaths@extraLayer
	# yeah, it is
	# does this work? let's find out
	pushd vendor/gitlab.com/elixxir/gpumaths/cgbnBindings/powm/
	make
	make install
	popd

update_master:
	GOFLAGS="" go get -u gitlab.com/elixxir/primitives@master
	GOFLAGS="" go get -u gitlab.com/elixxir/crypto@master
	GOFLAGS="" go get -u gitlab.com/elixxir/comms@master

master: update update_master build

release: update update_release build
