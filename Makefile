all: binary

docker: binary
	docker build .

binary:
	go build -v .

clean:
	rm -f procwatch

.PHONY: all docker binary clean

