// Package x stores utility functions, mostly for internal usage.
package x

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
)

var log = Log("x")

// Create and seed the generator.
// Typically a non-fixed seed should be used, such as time.Now().UnixNano().
// Using a fixed seed will produce the same output on every run.
var r = rand.New(rand.NewSource(time.Now().UnixNano()))

// Status stores any error codes retuned along with the error message; and
// is converted to JSON and returned if there's any error during the
// result.WriteJsonResponse call.
type Status struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Entity struct {
	Kind string
	Id   string
}

// Doc is the format data gets stored in search engine.
type Doc struct {
	Kind   string
	Id     string
	Values map[string]interface{}
	NanoTs int64
}

// Instruction is the format data gets stored in the underlying data stores.
type Instruction struct {
	SubjectId   string `json:"subject_id,omitempty"`
	SubjectType string `json:"subject_type,omitempty"`
	Predicate   string `json:"predicate,omitempty"`
	Object      []byte `json:"object,omitempty"`
	ObjectId    string `json:"object_id,omitempty"`
	NanoTs      int64  `json:"nano_ts,omitempty"`
	Source      string `json:"source,omitempty"`
}

// GobEncode converts Instruction to a byte array.
func (i *Instruction) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	enc := gob.NewEncoder(w)
	if err := enc.Encode(i.SubjectId); err != nil {
		return nil, err
	}
	if err := enc.Encode(i.SubjectType); err != nil {
		return nil, err
	}
	if err := enc.Encode(i.Predicate); err != nil {
		return nil, err
	}
	if err := enc.Encode(i.Object); err != nil {
		return nil, err
	}
	if err := enc.Encode(i.ObjectId); err != nil {
		return nil, err
	}
	if err := enc.Encode(i.NanoTs); err != nil {
		return nil, err
	}
	if err := enc.Encode(i.Source); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// GobDecode decodes Instruction from a byte array.
func (i *Instruction) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	dec := gob.NewDecoder(r)
	if err := dec.Decode(&i.SubjectId); err != nil {
		return err
	}
	if err := dec.Decode(&i.SubjectType); err != nil {
		return err
	}
	if err := dec.Decode(&i.Predicate); err != nil {
		return err
	}
	if err := dec.Decode(&i.Object); err != nil {
		return err
	}
	if err := dec.Decode(&i.ObjectId); err != nil {
		return err
	}
	if err := dec.Decode(&i.NanoTs); err != nil {
		return err
	}
	if err := dec.Decode(&i.Source); err != nil {
		return err
	}
	return nil
}

// Its is used for providing a sort interface to []Instruction.
type Its []Instruction

func (its Its) Len() int           { return len(its) }
func (its Its) Swap(i, j int)      { its[i], its[j] = its[j], its[i] }
func (its Its) Less(i, j int) bool { return its[i].NanoTs < its[j].NanoTs }

// Error constants.
const (
	E_ERROR            = "E_ERROR"
	E_INVALID_METHOD   = "E_INVALID_METHOD"
	E_INVALID_REQUEST  = "E_INVALID_REQUEST"
	E_INVALID_USER     = "E_INVALID_USER"
	E_MISSING_REQUIRED = "E_MISSING_REQUIRED"
	E_OK               = "E_OK"
	E_UNAUTHORIZED     = "E_UNAUTHORIZED"
)

// Log returns a logrus.Entry with a package field set.
func Log(p string) *logrus.Entry {
	logrus.SetLevel(logrus.DebugLevel)
	l := logrus.WithFields(logrus.Fields{
		"package": p,
	})
	return l
}

// LogErr returns a logrus.Entry with an error field set.
func LogErr(entry *logrus.Entry, err error) *logrus.Entry {
	return entry.WithFields(logrus.Fields{
		"error": err.Error(),
	})
}

// SetStatus creates, converts to JSON, and writes a Status object
// to http.ResponseWriter.
func SetStatus(w http.ResponseWriter, code, msg string) {
	r := &Status{Code: code, Message: msg}
	if js, err := json.Marshal(r); err == nil {
		fmt.Fprint(w, string(js))
	} else {
		panic(fmt.Sprintf("Unable to marshal: %+v", r))
	}
}

// ParseRequest parses a JSON based POST or PUT request into the provided
// Golang interface.
func ParseRequest(w http.ResponseWriter, r *http.Request, data interface{}) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&data); err != nil {
		SetStatus(w, E_ERROR, fmt.Sprintf("While parsing request: %v", err))
		return false
	}
	return true
}

const alphachars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// UniqueString generates a unique string only using the characters from
// alphachars constant, with length as specified.
func UniqueString(alpha int) string {
	var buf bytes.Buffer
	for i := 0; i < alpha; i++ {
		idx := r.Intn(len(alphachars))
		buf.WriteByte(alphachars[idx])
	}
	return buf.String()
}

// ParseIdFromUrl parses id from url (if it's a suffix) in this format:
//  url = host/xyz/id, urlToken = /xyz/ => uid = id
//  url = host/a/b/id, urlToken = /b/   => uid = id
func ParseIdFromUrl(r *http.Request, urlToken string) (uid string, ok bool) {
	url := r.URL.Path
	idx := strings.LastIndex(url, urlToken)
	if idx == -1 {
		return
	}
	return url[idx+len(urlToken):], true
}

// Reply would JSON marshal the provided rep Go interface object, and
// write that to http.ResponseWriter. In case of error, call SetStatus
// with the error.
func Reply(w http.ResponseWriter, rep interface{}) {
	if js, err := json.Marshal(rep); err == nil {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, string(js))
	} else {
		SetStatus(w, E_ERROR, err.Error())
	}
}
