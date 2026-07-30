module datastar-counter-sse

go 1.25.0

require (
	github.com/dhamidi/htmlc v0.0.0
	github.com/dhamidi/htmlc/hypermedia/datastar v0.0.0
	github.com/starfederation/datastar-go v1.2.2
)

require (
	github.com/CAFxX/httpcompression v0.0.9 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	golang.org/x/net v0.51.0 // indirect
)

replace github.com/dhamidi/htmlc => ../..

replace github.com/dhamidi/htmlc/hypermedia/datastar => ../../hypermedia/datastar
