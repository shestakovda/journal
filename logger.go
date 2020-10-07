package journal

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
)

type GlogLogger struct{}

func (l *GlogLogger) Print(tpl string, args ...interface{}) {
	glog.Infof(tpl, args...)
}

func (l *GlogLogger) Error(tpl string, args ...interface{}) {
	glog.Errorf(tpl, args...)
	glog.Flush()
}

type StrLogger struct {
	Result string

	buf strings.Builder
}

func (l *StrLogger) Print(tpl string, args ...interface{}) {
	l.buf.WriteString(fmt.Sprintf(tpl, args...) + "\n")
}

func (l *StrLogger) Error(tpl string, args ...interface{}) {
	l.Print(tpl, args...)
	l.Result += l.buf.String()
	l.buf.Reset()
}
