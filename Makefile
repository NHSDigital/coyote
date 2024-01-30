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
deps: test/functional/.venv/deps-installed venv

.PHONY: test
test: deps
	(cd coyoteadapters && go test)
	(cd coyotecore && go test)
	test/functional/.venv/bin/pytest -v test/functional

build/bin/coyote: $(shell find coyote* -type f -name '*.go')
	mkdir -p build/bin
	cd coyote && go build -o ../build/bin/coyote .

.PHONY: exe
exe: build/bin/coyote