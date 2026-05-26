package queue

import "testing"

func TestIsReconcileBusy(t *testing.T) {
	if !IsReconcileBusy(ErrReconcileBusy{RunID: "x"}) {
		t.Fatal("expected busy")
	}
	if IsReconcileBusy(nil) {
		t.Fatal("nil should not be busy")
	}
}
