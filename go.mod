module github.com/open-component-model/model-server

go 1.26.3

require (
	github.com/go-chi/chi/v5 v5.2.5
	github.com/go-playground/validator/v10 v10.30.2
	github.com/prometheus/client_golang v1.23.2
	github.com/stretchr/testify v1.11.1
	ocm.software/open-component-model/bindings/go/blob v0.0.13
	ocm.software/open-component-model/bindings/go/ctf v0.4.0
	ocm.software/open-component-model/bindings/go/descriptor/runtime v0.0.0-20260521121644-2be2fc986501
	ocm.software/open-component-model/bindings/go/descriptor/v2 v2.0.3-alpha3
	ocm.software/open-component-model/bindings/go/oci v0.0.43
	ocm.software/open-component-model/bindings/go/repository v0.0.9
	ocm.software/open-component-model/bindings/go/runtime v0.0.8
	sigs.k8s.io/yaml v1.6.0
)

require (
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cyberphone/json-canonicalization v0.0.0-20241213102144-19d51d7fe467 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nlepage/go-tarfs v1.2.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/veqryn/slog-context v0.9.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	oras.land/oras-go/v2 v2.6.0 // indirect
)

replace github.com/ThalesIgnite/crypto11 => github.com/ThalesGroup/crypto11 v1.2.4
