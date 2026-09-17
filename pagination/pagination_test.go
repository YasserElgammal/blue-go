package pagination_test

import (
	"reflect"
	"testing"

	"github.com/yasserelgammal/blue-go/pagination"
)

func TestMetadata(t *testing.T) {
	metadata := (pagination.Pagination{Page: 2, PerPage: 20, Total: 150}).Metadata()
	want := pagination.Metadata{CurrentPage: 2, PerPage: 20, Total: 150, LastPage: 8}
	if !reflect.DeepEqual(metadata, want) {
		t.Fatalf("metadata = %#v, want %#v", metadata, want)
	}
}

func TestMetadataNormalizesInvalidValues(t *testing.T) {
	metadata := (pagination.Pagination{Page: 0, PerPage: 0, Total: -5}).Metadata()
	want := pagination.Metadata{CurrentPage: 1, PerPage: 1, Total: 0, LastPage: 0}
	if !reflect.DeepEqual(metadata, want) {
		t.Fatalf("metadata = %#v, want %#v", metadata, want)
	}
}
