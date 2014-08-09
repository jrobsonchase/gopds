package gopds

import (
	"encoding/xml"
)

const (
	Nav byte = iota
	Acq
	Search
)

type OpdsFeed struct {
	XMLName xml.Name `xml:"feed"`
	*OpdsCommon
	XmlNs   string       `xml:"xmlns,attr,omitempty"`
	Entries []*OpdsEntry `xml:"entry,omitempty"`
}

type OpdsFeedDB struct {
	*OpdsCommon
	Desc    string
	User    string `json:",omitempty"`
	Sort    byte
	Entries []string `json:",omitempty"`
}

type OpdsCommon struct {
	Id      string      `xml:"id,omitempty"`
	Title   string      `xml:"title,omitempty"`
	Name	string      `xml:"-"`
	Type    byte        `xml:"-"`
	Links   []*OpdsLink `xml:"link,omitempty" json:",omitempty"`
	Updated string      `xml:"updated,omitempty"`
	Author  *OpdsAuthor `xml:"author,omitempty" json:",omitempty"`
}

type OpdsLink struct {
	Rel    string       `xml:"rel,attr,omitempty"`
	Href   string       `xml:"href,attr,omitempty"`
	Type   string       `xml:"type,attr,omitempty"`
	Prices []*OpdsPrice `xml:"http://opds-spec.org/2010/catalog price,omitempty" json:",omitempty"`
}

type OpdsPrice struct {
	CurrencyCode string `xml:"currencycode,attr,omitempty"`
	Price        string `xml:",chardata"`
}

type OpdsAuthor struct {
	Name string `xml:"name,omitempty"`
	Uri  string `xml:"uri,omitempty"`
}

type OpdsEntry struct {
	Id string `xml:"id,omitempty"`
	*OpdsMeta
	Updated  string       `xml:"updated,omitempty"`
	Category string       `xml:"category,omitempty" json:",omitempty"`
	Content  *OpdsContent `xml:"content,omitempty" json:",omitempty"`
	Links    []*OpdsLink  `xml:"link,omitempty" json:",omitempty"`
	Order	int	`xml:"-"`
}

type OpdsMeta struct {
	Title     string      `xml:"title,omitempty" json:",omitempty"`
	Author    *OpdsAuthor `xml:"author,omitempty" json:",omitempty"`
	Publisher string      `xml:"http://purl.org/dc/terms/ publisher,omitempty" json:",omitempty"`
	Issued    string      `xml:"http://purl.org/dc/terms/ issued,omitempty" json:",omitempty"`
	Lang      string      `xml:"http://purl.org/dc/terms/ language,omitempty" json:",omitempty"`
	Summary   string      `xml:"summary,omitempty" json:",omitempty"`
	Rights    string      `xml:"rights,omitempty" json:",omitempty"`
	Cover     bool        `xml:"-"`
	Thumb     bool        `xml:"-"`
	CoverType string      `xml:"-"`
	ThumbType string      `xml:"-"`
}

type OpdsContent struct {
	Type    string `xml:"type,attr,omitempty" json:",omitempty"`
	Content string `xml:",chardata" json:",omitempty"`
}
