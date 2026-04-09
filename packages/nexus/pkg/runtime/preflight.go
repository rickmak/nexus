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

		candidate := PreflightHardFail
		if macNestedVirtContext && check.Name == "nested_virt" {
			candidate = PreflightUnsupportedNested
		} else if check.Installable {
			candidate = PreflightInstallableMissing
		}

		status = higherPreflightStatus(status, candidate)
	}

	result.Status = status
	return result
}

func higherPreflightStatus(current, candidate PreflightStatus) PreflightStatus {
	if preflightStatusRank(candidate) > preflightStatusRank(current) {
		return candidate
	}
	return current
}

func preflightStatusRank(status PreflightStatus) int {
	switch status {
	case PreflightHardFail:
		return 3
	case PreflightUnsupportedNested:
		return 2
	case PreflightInstallableMissing:
		return 1
	default:
		return 0
	}
}
