# Process monitor - build and check commands.
#
#   make          build the program (./process-monitor)
#   make run      build it and start it
#   make check    everything that must pass before a delivery
#   make help     list all the targets
#
# The build tag tcell_minimal leaves out the part of the tcell library that
# could run the external program "infocmp" for unknown terminals. With it,
# the package os/exec is not even linked: the binary cannot run other
# programs. The price is that only xterm-like terminals are supported
# (xterm, tmux, vt100, vt220, ansi), which covers WSL and Windows Terminal.

BIN  := process-monitor
TAGS := tcell_minimal

# Our packages that must use nothing but the Go standard library: they read
# /proc, compute the figures and send the signals. Only ui may use tcell.
CORE := main.go proc monitor control

.PHONY: all build run test race cover fmt vet deps check clean help

all: build

build:
	go build -tags $(TAGS) -o $(BIN) .

run: build
	./$(BIN)

test:
	go test -tags $(TAGS) ./...

race:
	go test -tags $(TAGS) -race ./...

cover:
	go test -tags $(TAGS) -cover ./...

fmt:
	@files="`gofmt -l .`"; \
	if [ -n "$$files" ]; then echo "Not formatted (run gofmt -w .):"; echo "$$files"; exit 1; fi; \
	echo "OK: all files are formatted"

vet:
	go vet -tags $(TAGS) ./...

# Checks the two rules about dependencies.
deps:
	@if grep -rlE 'gdamore/tcell|golang\.org/x/|"os/exec"|gopsutil' --include=*.go $(CORE); then \
		echo "FAIL: the files above use a library outside the standard one"; exit 1; \
	fi; \
	echo "OK: proc, monitor, control and main.go use only the Go standard library"
	@if go list -tags $(TAGS) -deps . | grep -qx 'os/exec'; then \
		echo "FAIL: os/exec is linked into the binary"; exit 1; \
	fi; \
	echo "OK: os/exec is not linked, so the binary cannot run external programs"

check: fmt vet deps test
	@echo "All checks passed"

clean:
	rm -f $(BIN)

help:
	@echo "make          build ./$(BIN) (with -tags $(TAGS))"
	@echo "make run      build and start the monitor"
	@echo "make test     run the tests"
	@echo "make race     run the tests with the race detector"
	@echo "make cover    run the tests and show the coverage"
	@echo "make deps     check that no forbidden library is used"
	@echo "make check    format + vet + deps + tests"
	@echo "make clean    remove the binary"
