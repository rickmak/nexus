package runtime

type PreflightStatus string

const (
	PreflightPass               PreflightStatus = "pass"
	PreflightInstallableMissing PreflightStatus = "installable_missing"
	PreflightUnsupportedNested  PreflightStatus = "unsupported_nested_virt"
	PreflightHardFail           PreflightStatus = "hard_fail"
)

type PreflightCheck struct {
	Name        string `json:"name"`
	OK          bool   `json:"ok"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
	Installable bool   `json:"installable,omitempty"`
}

type FirecrackerPreflightResult struct {
	Status         PreflightStatus  `json:"status"`
	Checks         []PreflightCheck `json:"checks"`
	SetupAttempted bool             `json:"setupAttempted"`
	SetupOutcome   string           `json:"setupOutcome"`
}

func ClassifyFirecrackerPreflight(checks []PreflightCheck, macNestedVirtContext bool) FirecrackerPreflightResult {
	result := FirecrackerPreflightResult{Checks: checks}

	status := PreflightPass
	for _, check := range checks {
		if check.OK {
			continue
		}

		if macNestedVirtContext && check.Name == "nested_virt" {
			status = PreflightUnsupportedNested
			break
		}

		if check.Installable {
			if status == PreflightPass {
				status = PreflightInstallableMissing
			}
			continue
		}

		status = PreflightHardFail
	}

	result.Status = status
	return result
}
