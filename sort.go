package gopds

import (
	"time"
)

type EntryComp func(*OpdsEntry,*OpdsEntry) bool

type EntrySorter struct {
	entries []*OpdsEntry
	compFunc EntryComp
}

func NewEntrySorter(entries []*OpdsEntry,compFunc EntryComp) *EntrySorter {
	return &EntrySorter{entries,compFunc}
}

func (s *EntrySorter) Less(i,j int) bool {
	return s.compFunc(s.entries[i],s.entries[j])
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
)

var (
	sortFuncBytes map[byte]EntryComp = map[byte]EntryComp{
		SortTitle: SortTitleFunc,
		SortAuthor: SortAuthorFunc,
		SortUpdated: SortUpdatedFunc}
	sortFuncStrings map[string]EntryComp = map[string]EntryComp{
		"title": SortTitleFunc,
		"author": SortAuthorFunc,
		"updated": SortUpdatedFunc}
)

func SortAuthorFunc(i,j *OpdsEntry) bool {
	iName := i.Author.Name
	jName := j.Author.Name
	return iName < jName
}

func SortTitleFunc(i,j *OpdsEntry) bool {
	iName := i.Title
	jName := j.Title
	return iName < jName
}

func SortUpdatedFunc(i,j *OpdsEntry) bool {
	iTime,_ := time.Parse(i.Updated,time.RFC3339)
	jTime,_ := time.Parse(j.Updated,time.RFC3339)
	return jTime.After(iTime)
}
