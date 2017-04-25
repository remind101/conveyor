// +build !newrelic_enabled

package newrelic

type NRTxReporter struct{}

func (r *NRTxReporter) ReportError(txnID int64, exceptionType, errorMessage, stackTrace, stackFrameDelim string) (int, error) {
	return 0, nil
}

func (r *NRTxReporter) ReportCustomMetric(name string, value float64) (int, error) {
	return 0, nil
}
