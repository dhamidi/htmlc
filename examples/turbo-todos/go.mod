module turbo-todos

go 1.25.0

require (
	github.com/dhamidi/htmlc v0.0.0
	github.com/dhamidi/htmlc/hypermedia/turbo v0.0.0
)

require golang.org/x/net v0.51.0 // indirect

replace github.com/dhamidi/htmlc => ../..

replace github.com/dhamidi/htmlc/hypermedia/turbo => ../../hypermedia/turbo
