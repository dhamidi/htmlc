.PHONY: htmlc v-syntax-highlight all

all: htmlc v-syntax-highlight

htmlc:
	go build -o bin/htmlc ./cmd/htmlc

v-syntax-highlight:
	cd cmd/v-syntax-highlight && go build -o ../../bin/v-syntax-highlight .
