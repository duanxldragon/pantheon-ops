package common

const (
	StatusEnabled  = 1
	StatusDisabled = 2
)

const (
	StatusFlagNo  = 0
	StatusFlagYes = 1
)

const (
	OperationStatusSuccess = 1
	OperationStatusFailure = 2
)

const (
	LoginStatusFailure = 0
	LoginStatusSuccess = 1
)

const (
	SessionStatusActive  = 1
	SessionStatusRevoked = 2
)

func IsEnabledStatus(status int) bool {
	return status == StatusEnabled || status == StatusDisabled
}

func NormalizeEnabledStatus(status int) int {
	if status == StatusDisabled {
		return StatusDisabled
	}
	return StatusEnabled
}

func IsLoginStatus(status int) bool {
	return status == LoginStatusSuccess || status == LoginStatusFailure
}
