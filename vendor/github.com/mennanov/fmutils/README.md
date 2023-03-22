# Golang protobuf FieldMask utils

[![Build Status](https://cloud.drone.io/api/badges/mennanov/fmutils/status.svg?ref=refs/heads/main)](https://cloud.drone.io/mennanov/fmutils)
[![Coverage Status](https://codecov.io/gh/mennanov/fmutils/branch/main/graph/badge.svg)](https://codecov.io/gh/mennanov/fmutils)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/mennanov/fmutils)](https://pkg.go.dev/github.com/mennanov/fmutils)

### Filter a protobuf message with a FieldMask applied

```go
// Keeps the fields mentioned in the paths untouched, all the other fields will be cleared.
fmutils.Filter(protoMessage, []string{"a.b.c", "d"})
```

### Prune a protobuf message with a FieldMask applied

```go
// Clears all the fields mentioned in the paths, all the other fields will be left untouched.
fmutils.Prune(protoMessage, []string{"a.b.c", "d"})
```

### Working with Golang protobuf APIv1

This library uses the [new Go API for protocol buffers](https://blog.golang.org/protobuf-apiv2).
If your `*.pb.go` files are generated with the old version APIv1 then you have 2 choices:

- migrate to the new APIv2 `google.golang.org/protobuf`
- upgrade an existing APIv1 version to `github.com/golang/protobuf@v1.4.0` that implements the new API

In both cases you'll need to regenerate `*.pb.go` files.

If you decide to stay with APIv1 then you need to use the [`proto.MessageV2`](https://pkg.go.dev/github.com/golang/protobuf@v1.4.3/proto#MessageV2) function like this:

```go
import protov1 "github.com/golang/protobuf/proto"

fmutils.Filter(protov1.MessageV2(protoMessage), []string{"a.b.c", "d"})
```

[Read more about the Go protobuf API versions.](https://blog.golang.org/protobuf-apiv2#TOC_4.)

### Examples

See the [examples_test.go](https://github.com/mennanov/fmutils/blob/main/examples_test.go) for real life examples.
