package main

type Set map[string]struct{}

func NewSet(items ...string) Set {
	set := make(Set)
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}

func (s Set) Has(value string) bool {
	_, ok := s[value]
	return ok
}
