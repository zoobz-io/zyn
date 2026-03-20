module github.com/zoobz-io/zyn/gemini

go 1.24

toolchain go1.25.3

replace github.com/zoobz-io/zyn => ../

require (
	github.com/zoobz-io/capitan v1.0.2
	github.com/zoobz-io/zyn v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/zoobz-io/clockz v1.0.2 // indirect
	github.com/zoobz-io/pipz v1.0.5 // indirect
	github.com/zoobz-io/sentinel v1.0.4 // indirect
)
