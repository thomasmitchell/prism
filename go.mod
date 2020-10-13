module github.com/thomasmitchell/prism

go 1.14

require (
	github.com/concourse/concourse v4.2.3+incompatible
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/concourse/concourse => github.com/concourse/concourse v1.6.1-0.20201002165707-b5584f13bfe7
