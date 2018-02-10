package main

import (
	"fmt"
	"reflect"
)

// SearchTerm is search term in GitHub object
// See: https://developer.github.com/v3/search/#parameters-2
type SearchTerm struct {
	language  string
	size      string
	path      string
	filename  string
	extension string
	user      string
	repo      string
}

// NewSearchTerm creates SearchTerm
func NewSearchTerm() *SearchTerm {
	return &SearchTerm{}
}

func (s *SearchTerm) debugf() {
	v := reflect.Indirect(reflect.ValueOf(s))
	t := v.Type()

	var name, value string
	for i := 0; i < t.NumField(); i++ {
		name = t.Field(i).Name
		value = v.Field(i).String()
		Debugf("%s: %s", name, value)
	}
}

func (s *SearchTerm) query(keyword string) string {
	q := keyword
	v := reflect.Indirect(reflect.ValueOf(s))
	t := v.Type()

	var name, value string
	for i := 0; i < t.NumField(); i++ {
		name = t.Field(i).Name
		value = v.Field(i).String()

		if value != "" {
			q = fmt.Sprintf("%s %s:%s", q, name, value)
		}
	}
	return q
}
