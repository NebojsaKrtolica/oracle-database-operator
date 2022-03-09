package ociutil

import (
	"time"

	"github.com/oracle/oci-go-sdk/v54/common"
)

// Follow the format of the Display Name
const displayFormat = "2006-01-02 15:04:05 MST"

func FormatSDKTime(dateTime time.Time) string {
	return dateTime.Format(displayFormat)
}

func ParseDisplayTime(val string) (*common.SDKTime, error) {
	parsedTime, err := time.Parse(displayFormat, val)
	if err != nil {
		return nil, err
	}
	sdkTime := common.SDKTime{Time: parsedTime}
	return &sdkTime, nil
}