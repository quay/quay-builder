package main

import log "github.com/sirupsen/logrus"

// realmFormatter implements the logrus.Formatter interface such that all
// log messages are prefixed with the realm of the build.
type realmFormatter struct {
	realm     string
	formatter log.Formatter
}

func newRealmFormatter(realm string) log.Formatter {
	return &realmFormatter{
		realm:     realm,
		formatter: new(log.TextFormatter),
	}
}

func (f *realmFormatter) Format(entry *log.Entry) ([]byte, error) {
	if f.realm != "" {
		entry.Message = f.realm + ": " + entry.Message
	}
	return f.formatter.Format(entry)
}
