package domain

import (
	"fmt"
	"regexp"
)

const (
	FromMethod   string = "from"
	ToMethod            = "to"
	IntoMethod          = "into"
	UpdateMethod        = "update"
	DeleteMethod        = "delete"
)

type Query struct {
	Use        Modifiers
	Statements []Statement
}

type Modifiers map[string]interface{}

type Statement struct {
	Method       string
	Resource     string
	Alias        string
	In           []string
	Headers      map[string]interface{}
	Timeout      interface{}
	With         Params
	Only         []interface{}
	Hidden       bool
	CacheControl CacheControl
	IgnoreErrors bool
}

type Params struct {
	Body   interface{}
	Values map[string]interface{}
}

type CacheControl struct {
	MaxAge  interface{}
	SMaxAge interface{}
}

type Variable struct {
	Target string
}

type Chain []interface{}

type Flatten struct {
	Target interface{}
}

type Json struct {
	Target interface{}
}

type Base64 struct {
	Target interface{}
}

type Match struct {
	Target []string
	Arg    *regexp.Regexp
}

type QueryOptions struct {
	Namespace string
	Id        string
	Revision  int
	Tenant    string
}

type QueryInput struct {
	Params  map[string]interface{}
	Headers map[string]string
}

type QueryContext struct {
	Mappings map[string]Mapping
	Options  QueryOptions
	Input    QueryInput
}

type SavedQuery struct {
	Text       string
	Deprecated bool
}

type ErrQueryRevisionDeprecated struct {
	Revision int
}

func (e ErrQueryRevisionDeprecated) Error() string {
	return fmt.Sprintf("the revision %d is deprecated", e.Revision)
}
