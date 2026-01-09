package cmd

import (
	"net/http"
	"testing"

	"google.golang.org/api/googleapi"
)

func TestResolveLabelIDs_Extra(t *testing.T) {
	ids := resolveLabelIDs([]string{"  Foo ", "bar"}, map[string]string{"foo": "id1"})
	if len(ids) != 2 || ids[0] != "id1" || ids[1] != "bar" {
		t.Fatalf("unexpected ids: %v", ids)
	}
}

func TestLabelDuplicateChecks(t *testing.T) {
	if !labelAlreadyExistsMessage("Label name exists") {
		t.Fatalf("expected label exists")
	}
	if !labelDuplicateReason("duplicate") {
		t.Fatalf("expected duplicate reason")
	}

	err := &googleapi.Error{Code: http.StatusConflict, Message: "label already exists"}
	if !isDuplicateLabelError(err) {
		t.Fatalf("expected duplicate label error")
	}
}
