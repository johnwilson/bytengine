package filesystem

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nu7hatch/gouuid"
)

func ValidateDbName(d string) error {
	d = strings.ToLower(d)

	// regex verification
	r, err := regexp.Compile("^[a-z][a-z0-9_]{1,20}$")
	if err != nil {
		return err
	}
	if r.MatchString(d) {
		return nil
	}
	msg := fmt.Sprintf("database name '%s' isn't valid.", d)
	return errors.New(msg)
}

func ValidateDirName(d string) error {
	msg := fmt.Sprintf("directory name '%s' isn't valid.", d)
	r, err := regexp.Compile("^[a-zA-Z0-9][a-zA-Z0-9_\\-]{0,}$")
	if err != nil {
		return errors.New(msg)
	}
	if r.MatchString(d) {
		return nil
	}
	return errors.New(msg)
}

func ValidateCounterName(c string) error {
	msg := fmt.Sprintf("counter name '%s' isn't valid.", c)
	r, err := regexp.Compile("[a-zA-Z0-9_\\.\\-]+")
	if err != nil {
		return errors.New(msg)
	}
	match := r.FindString(c)
	if match != c {
		return errors.New(msg)
	}
	return nil
}

func ValidateFileName(f string) error {
	msg := fmt.Sprintf("file name '%s' isn't valid.", f)
	r, err := regexp.Compile("^\\w[\\w\\-]{0,}(\\.[a-zA-Z0-9]+)*$")
	if err != nil {
		return errors.New(msg)
	}
	if r.MatchString(f) {
		return nil
	}
	return errors.New(msg)
}

func FormatDatetime(t time.Time) string {
	f := "%d:%02d:%02d-%02d:%02d:%02d.%03d"
	dt := fmt.Sprintf(f,
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond()/100000)
	return dt
}

func NewNodeID() (string, error) {
	tmp, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	id := strings.Replace(tmp.String(), "-", "", -1) // remove dashes
	return id, nil
}
