package tracking

import (
	"testing"

	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/opendatahub-io/mlflow-go/internal/gen/mlflowpb"
)

// TestNoOTelProtoRegistryConflict guards against reintroducing a generated
// stub for opentelemetry/proto/trace/v1/trace.proto.
//
// The MLflow service.proto references opentelemetry.proto.trace.v1.Span. If we
// generate our own Go package for that proto path instead of mapping it to the
// official OTel types, any binary importing both mlflow-go and the OTel SDK
// panics at init with "file ... is already registered".
//
// Importing both packages in this test binary is most of the check: a
// regression panics before any test runs. The assertions below pin down the
// reason, so a failure reads as more than an unexplained init panic.
func TestNoOTelProtoRegistryConflict(t *testing.T) {
	const path = "opentelemetry/proto/trace/v1/trace.proto"

	fd, err := protoregistry.GlobalFiles.FindFileByPath(path)
	if err != nil {
		t.Fatalf("FindFileByPath(%q): %v", path, err)
	}

	// The official package declares the full Span message; a stub would not.
	if got := fd.Path(); got != path {
		t.Errorf("descriptor path = %q, want %q", got, path)
	}

	want := (&otlptrace.Span{}).ProtoReflect().Descriptor().ParentFile()
	if fd != want {
		t.Errorf("registered %q resolves to a descriptor other than the one from "+
			"go.opentelemetry.io/proto/otlp/trace/v1; a generated stub has likely "+
			"been reintroduced", path)
	}

	// Trace.spans must be typed by the official Span, not a local stub.
	spans := (&mlflowpb.Trace{}).ProtoReflect().Descriptor().Fields().ByName("spans")
	if spans == nil {
		t.Fatal("mlflowpb.Trace has no spans field")
	}

	if got := spans.Message().ParentFile(); got != want {
		t.Errorf("Trace.spans message resolves to %q, want the official OTel descriptor",
			got.Path())
	}
}
