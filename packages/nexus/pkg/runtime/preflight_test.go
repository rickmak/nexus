package runtime

import "testing"

func TestClassifyFirecrackerPreflight_Statuses(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		checks := []PreflightCheck{
			{Name: "nested_virt", OK: true},
			{Name: "lima", OK: true},
			{Name: "tap", OK: true},
		}

		got := ClassifyFirecrackerPreflight(checks, false)
		if got.Status != PreflightPass {
			t.Fatalf("expected pass, got %s", got.Status)
		}
	})

	t.Run("installable missing", func(t *testing.T) {
		checks := []PreflightCheck{
			{Name: "lima", OK: false, Installable: true},
		}

		got := ClassifyFirecrackerPreflight(checks, false)
		if got.Status != PreflightInstallableMissing {
			t.Fatalf("expected installable_missing, got %s", got.Status)
		}
	})

	t.Run("hard fail", func(t *testing.T) {
		checks := []PreflightCheck{
			{Name: "lima", OK: false, Installable: true},
			{Name: "kernel_support", OK: false},
		}

		got := ClassifyFirecrackerPreflight(checks, false)
		if got.Status != PreflightHardFail {
			t.Fatalf("expected hard_fail, got %s", got.Status)
		}
	})

	t.Run("unsupported nested virt", func(t *testing.T) {
		checks := []PreflightCheck{{Name: "nested_virt", OK: false, Message: "nested virtualization unsupported"}}

		got := ClassifyFirecrackerPreflight(checks, true)
		if got.Status != PreflightUnsupportedNested {
			t.Fatalf("expected unsupported_nested_virt, got %s", got.Status)
		}
	})
}

func TestClassifyFirecrackerPreflight_UnsupportedNestedVirt(t *testing.T) {
	checks := []PreflightCheck{{Name: "nested_virt", OK: false, Message: "nested virtualization unsupported"}}

	got := ClassifyFirecrackerPreflight(checks, true)
	if got.Status != PreflightUnsupportedNested {
		t.Fatalf("expected unsupported_nested_virt, got %s", got.Status)
	}
}
