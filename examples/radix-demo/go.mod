module radix-demo

go 1.25.0

require (
	github.com/dhamidi/htmlc v0.0.0
	github.com/dhamidi/htmlc/ui/radix v0.0.0
)

require golang.org/x/net v0.51.0 // indirect

replace github.com/dhamidi/htmlc => ../..

replace github.com/dhamidi/htmlc/ui/radix => ../../ui/radix
