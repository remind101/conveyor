// +build newrelic_enabled

package newrelic

import (
	"github.com/remind101/newrelic/sdk"
)

type NRTxReporter struct{}

func (r *NRTxReporter) ReportError(txnID int64, exceptionType, errorMessage, stackTrace, stackFrameDelim string) (int, error) {
	return sdk.TransactionNoticeError(txnID, exceptionType, errorMessage, stackTrace, stackFrameDelim)
}

func (r *NRTxReporter) ReportCustomMetric(name string, value float64) (int, error) {
	return sdk.RecordMetric(name, value)
}
