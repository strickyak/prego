# github.com/strickyak/prego/Makefile

all:
	go build
	go test
	cd tests; make
