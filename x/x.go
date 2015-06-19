package x

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
)

var log = Log("x")

type Status struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	NOOP                      = 0
	ADD                       = 1
	REMOVE                    = 10
	REPLACE_ONE_PER_PREDICATE = 20
	REPLACE_ONE_PER_SOURCE    = 21
)

type Node struct {
	Name      string            `json:"name,omitempty"`
	Type      string            `json:"type,omitempty"`
	Id        string            `json:"id,omitempty"`
	Timestamp time.Time         `json:"timestamp,omitempty"`
	Source    string            `json:"source,omitempty"`
	Edges     map[string][]Node `json:"edges,omitempty"`
}

type Instruction struct {
	SubjectId   string `json:"subject_id,omitempty"`
	SubjectType string `json:"subject_type,omitempty"`
	Predicate   string `json:"predicate,omitempty"`
	ObjectText  string `json:"object_text,omitempty"`
	ObjectId    string `json:"object_id,omitempty"`
	NanoTs      int64  `json:"nano_ts,omitempty"`
	Source      string `json:"source,omitempty"`
	Operation   int    `json:"operation,omitempty"`
}

const (
	E_ERROR            = "E_ERROR"
	E_INVALID_METHOD   = "E_INVALID_METHOD"
	E_INVALID_REQUEST  = "E_INVALID_REQUEST"
	E_INVALID_USER     = "E_INVALID_USER"
	E_MISSING_REQUIRED = "E_MISSING_REQUIRED"
	E_OK               = "E_OK"
	E_UNAUTHORIZED     = "E_UNAUTHORIZED"
)

func Log(p string) *logrus.Entry {
	logrus.SetLevel(logrus.DebugLevel)
	l := logrus.WithFields(logrus.Fields{
		"package": p,
	})
	return l
}

func LogErr(entry *logrus.Entry, err error) *logrus.Entry {
	return entry.WithFields(logrus.Fields{
		"error": err.Error(),
	})
}

func SetStatus(w http.ResponseWriter, code, msg string) {
	r := &Status{Code: code, Message: msg}
	if js, err := json.Marshal(r); err == nil {
		fmt.Fprint(w, string(js))
	} else {
		panic(fmt.Sprintf("Unable to marshal: %+v", r))
	}
}

func ParseRequest(w http.ResponseWriter, r *http.Request, data interface{}) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&data); err != nil {
		SetStatus(w, E_ERROR, fmt.Sprintf("While parsing request: %v", err))
		return false
	}
	return true
}

const alphachars = "0123456789abcdefghijklmnopqrstuvwxyz"

func UniqueString(alpha int) string {
	var buf bytes.Buffer
	for i := 0; i < alpha; i++ {
		idx := rand.Intn(len(alphachars))
		buf.WriteByte(alphachars[idx])
	}
	return buf.String()
}
