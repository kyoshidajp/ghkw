package main

import (
	"fmt"
)

// SearchTerm is search term in GitHub object
type SearchTerm struct {
	language  string
	filename  string
	extension string
}

// NewSearchTerm creates SearchTerm
func NewSearchTerm(language string, filename string, extension string) *SearchTerm {
	Debugf("language: %s", language)
	Debugf("filename: %s", filename)
	Debugf("extension: %s", extension)

	return &SearchTerm{
		language:  language,
		filename:  filename,
		extension: extension,
	}
}

func (s *SearchTerm) query(keyword string) string {
	q := keyword
	if s.language != "" {
		q = fmt.Sprintf("%s language:%s", q, s.language)
	}
	if s.filename != "" {
		q = fmt.Sprintf("%s filename:%s", q, s.filename)
	}
	if s.extension != "" {
		q = fmt.Sprintf("%s extension:%s", q, s.extension)
	}
	return q
}
