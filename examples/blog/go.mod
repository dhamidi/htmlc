module blog

go 1.25.0

require github.com/dhamidi/htmlc v0.0.0

require (
	github.com/yuin/goldmark v1.7.4
	github.com/yuin/goldmark-highlighting/v2 v2.0.0-20230729083705-37449abec8cc
)

require (
	github.com/alecthomas/chroma/v2 v2.2.0 // indirect
	github.com/dlclark/regexp2 v1.7.0 // indirect
	golang.org/x/net v0.51.0 // indirect
)

replace github.com/dhamidi/htmlc => ../..
