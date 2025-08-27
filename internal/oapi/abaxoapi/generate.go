package abaxoapi

//go:generate echo applying overlay...
//go:generate sh -c "go tool -modfile ../../../tools/go.mod openapi-overlay apply overlay.yaml 01-original.json > 02-overlayed.json"

//go:generate echo generating code...
//go:generate go tool -modfile ../../../tools/go.mod oapi-codegen -config config.yaml 02-overlayed.json
