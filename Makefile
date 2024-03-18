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

.PHONY: test
test: deps
	test/functional/.venv/bin/pytest -v test/functional

build/bin/coyote: $(shell find . -type f -name '*.go')
	mkdir -p build/bin
	go build -o build/bin/coyote ./cmd/coyote/main.go

.PHONY: exe
exe: build/bin/coyote

.PHONY: home-install
home-install: build/bin/coyote
	mkdir -p ~/bin # This might not be on $$PATH, so check that
	cp build/bin/coyote ~/bin/coyote