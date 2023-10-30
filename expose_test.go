package apig

import "github.com/SpalkLtd/slogger"

func SetLogger(testLogger *slogger.SpalkLogger) {
	logger = testLogger
}
