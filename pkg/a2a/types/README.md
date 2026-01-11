# A2A Generated Types

This package hosts Go types generated from `pkg/a2a/proto/a2a.proto`.

Generation:
```bash
./scripts/gen-a2a.sh
```

Notes:
- Install `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` before running.
- Provide googleapis protos and set `A2A_GOOGLEAPIS_DIR` (expects `google/api/annotations.proto`).
- The proto file is the normative source; do not edit generated files manually.
