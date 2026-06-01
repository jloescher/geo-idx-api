package repository

import (
	"reflect"
	"testing"
)

func TestEncodeDecodeGrantedMCPKeyIDs(t *testing.T) {
	ids := []int64{10, 20, 30}
	raw, err := encodeGrantedMCPKeyIDs(ids)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "[10,20,30]" {
		t.Fatalf("unexpected json: %s", raw)
	}
	var decoded []int64
	if err := decodeGrantedMCPKeyIDs(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, ids) {
		t.Fatalf("decoded=%v want=%v", decoded, ids)
	}
}

func TestEncodeGrantedMCPKeyIDsNilSlice(t *testing.T) {
	raw, err := encodeGrantedMCPKeyIDs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "[]" {
		t.Fatalf("nil slice should encode as empty array, got %s", raw)
	}
}
