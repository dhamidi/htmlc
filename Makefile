.PHONY: htmlc v-syntax-highlight all docs

all: htmlc v-syntax-highlight

htmlc:
	go build -o bin/htmlc ./cmd/htmlc

v-syntax-highlight:
	cd cmd/v-syntax-highlight && go build -o ../../bin/v-syntax-highlight .

# docs builds the documentation site under site/out/.
# v-syntax-highlight is copied into the component directory so that
# htmlc can discover and run it during the build.
docs: htmlc v-syntax-highlight
	cp bin/v-syntax-highlight site/v-syntax-highlight
	chmod +x site/v-syntax-highlight
	mkdir -p site/assets
	bin/v-syntax-highlight -print-css -style monokai > site/assets/highlight.css
	bin/htmlc build -dir site -pages site/pages -out site/out -layout Layout
