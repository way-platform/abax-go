package abaxoapi

//go:generate echo applying overlay...
//go:generate sh -c "openapi-overlay apply overlay.yaml 01-original.json > 02-overlayed.json"

//go:generate echo generating code...
//go:generate oapi-codegen -config config.yaml 02-overlayed.json
