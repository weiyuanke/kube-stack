package utils

import (
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

// ExtractLabelSelector extract labelSelctor from string
// Parse takes a string representing a selector and returns a selector
// object, or an error. This parsing function differs from ParseSelector
// as they parse different selectors with different syntaxes.
// The input will cause an error if it does not follow this form:
//
//	<selector-syntax>         ::= <requirement> | <requirement> "," <selector-syntax>
//	<requirement>             ::= [!] KEY [ <set-based-restriction> | <exact-match-restriction> ]
//	<set-based-restriction>   ::= "" | <inclusion-exclusion> <value-set>
//	<inclusion-exclusion>     ::= <inclusion> | <exclusion>
//	<exclusion>               ::= "notin"
//	<inclusion>               ::= "in"
//	<value-set>               ::= "(" <values> ")"
//	<values>                  ::= VALUE | VALUE "," <values>
//	<exact-match-restriction> ::= ["="|"=="|"!="] VALUE
//
// KEY is a sequence of one or more characters following [ DNS_SUBDOMAIN "/" ] DNS_LABEL. Max length is 63 characters.
// VALUE is a sequence of zero or more characters "([A-Za-z0-9_-\.])". Max length is 63 characters.
// Delimiter is white space: (' ', '\t')
// Example of valid syntax:
//
//	"x in (foo,,baz),y,z notin ()"
func ExtractLabelSelector(l string) (labels.Selector, error) {
	s, err := labels.Parse(l)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func ExtractFieldSelector(f string) (fields.Selector, error) {
	fieldSelector, err := fields.ParseSelector(f)
	if err != nil {
		return nil, err
	}
	return fieldSelector, nil
}
