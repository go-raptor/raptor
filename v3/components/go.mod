module github.com/go-raptor/raptor/v3/components

go 1.24.0

replace github.com/go-raptor/raptor/v3/config => ../config

require (
	github.com/go-raptor/connector v1.0.1
	github.com/go-raptor/errs v1.0.0
	github.com/go-raptor/raptor/v3/config v0.0.0-00010101000000-000000000000
	github.com/labstack/echo/v4 v4.13.3
	github.com/pwntr/tinter v1.1.2
)

require (
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
