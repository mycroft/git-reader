all: clean build

build:
	go build

clean:
	rm -f git-reader

run: build
	./git-reader

.PHONY: all build clean
