default: exe

test/functional/.venv/bin/activate:
	python3 -m venv test/functional/.venv

.PHONY: venv
venv: test/functional/.venv/bin/activate

test/functional/.venv/deps-installed:
	test/functional/.venv/bin/pip install -r test/functional/requirements.txt
	touch test/functional/.venv/deps-installed

.PHONY: sh
sh: deps
	source test/functional/.venv/bin/activate && exec $$SHELL

.PHONY: deps
deps: venv test/functional/.venv/deps-installed

.PHONY: test-unit
test-unit:
	go test -v ./...

.PHONY: test-functional
test-functional: exe deps
	test/functional/.venv/bin/pytest -v test/functional

.PHONY: test
test: test-unit test-functional

build/bin/coyote: $(shell find . -type f -name '*.go')
	mkdir -p build/bin
	go build -o build/bin/coyote ./cmd/coyote/main.go

.PHONY: exe
exe: build/bin/coyote

.PHONY: home-install
home-install: build/bin/coyote
	mkdir -p ~/bin # This might not be on $$PATH, so check that
	cp build/bin/coyote ~/bin/coyote

build/%.tgz: build/bin-%/coyote
	mkdir -p build/packages
	tar -czf build/$*.tgz -C build/bin-$* coyote

.PHONY: packages
packages: build/linux-amd64.tgz
packages: build/linux-arm64.tgz
packages: build/darwin-amd64.tgz
packages: build/darwin-arm64.tgz

build/bin-%/coyote: $(shell find . -type f -name '*.go')
	mkdir -p build/bin-$*
	export GOOS=$$(echo $* | cut -d- -f1)
	export GOARCH=$$(echo $* | cut -d- -f2)
	go build -o build/bin-$*/coyote ./cmd/coyote/main.go


.ONESHELL: