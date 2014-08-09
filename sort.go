package gopds

import (
	"time"
)

const (
	lt byte = iota
	gt
	eq
)

type EntryComp func(*OpdsEntry,*OpdsEntry) byte

type EntrySorter struct {
	entries []*OpdsEntry
	compFunc EntryComp
}

func NewEntrySorter(entries []*OpdsEntry,compFunc EntryComp) *EntrySorter {
	return &EntrySorter{entries,compFunc}
}

func (s *EntrySorter) Less(i,j int) bool {
	switch s.compFunc(s.entries[i],s.entries[j]) {
	case lt,eq:
		return true
	default:
		return false
	}
}

func (s *EntrySorter) Swap(i,j int) {
	tmp := s.entries[i]
	s.entries[i] = s.entries[j]
	s.entries[j] = tmp
}

func (s *EntrySorter) Len() int {
	return len(s.entries)
}

const (
	SortTitle byte = iota
	SortAuthor
	SortUpdated
	SortOrder
)

var (
	sortFuncBytes map[byte]EntryComp = map[byte]EntryComp{
		SortTitle: SortTitleFunc,
		SortAuthor: SortAuthorFunc,
		SortUpdated: SortUpdatedFunc,
		SortOrder: SortOrderFunc}
	sortFuncStrings map[string]EntryComp = map[string]EntryComp{
		"title": SortTitleFunc,
		"author": SortAuthorFunc,
		"updated": SortUpdatedFunc}
)

func SortAuthorFunc(i,j *OpdsEntry) byte {
	iName := i.Author.Name
	jName := j.Author.Name
	if iName == jName {
		return eq
	} else if iName < jName {
		return lt
	}
	return gt
}

func SortTitleFunc(i,j *OpdsEntry) byte {
	iName := i.Title
	jName := j.Title
	if iName == jName {
		return eq
	} else if iName < jName {
		return lt
	}
	return gt
}

func SortUpdatedFunc(i,j *OpdsEntry) byte {
	iTime,_ := time.Parse(i.Updated,time.RFC3339)
	jTime,_ := time.Parse(j.Updated,time.RFC3339)
	if jTime.After(iTime) {
		return lt
	}
	return gt
}

func SortOrderFunc(i,j *OpdsEntry) byte {
	iName := i.Order
	jName := j.Order
	if iName == jName {
		return eq
	} else if iName < jName {
		return lt
	}
	return gt
}

func SortCompose(funcs... EntryComp) EntryComp {
	return func(i *OpdsEntry,j *OpdsEntry) byte {
		for _,v := range funcs {
			cmp := v(i,j)
			switch cmp {
			case lt,gt:
				return cmp
			}
		}
		return eq
	}
}


