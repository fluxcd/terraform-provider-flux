package utils

type DiscardLogger struct {
}

func (DiscardLogger) Actionf(format string, a ...interface{}) {

}

func (DiscardLogger) Generatef(format string, a ...interface{}) {

}

func (DiscardLogger) Waitingf(format string, a ...interface{}) {

}

func (DiscardLogger) Successf(format string, a ...interface{}) {

}

func (DiscardLogger) Warningf(format string, a ...interface{}) {

}

func (DiscardLogger) Failuref(format string, a ...interface{}) {

}
