package gopds

import (
	"encoding/xml"
)

const (
	Nav byte = iota
	Acq
)

type OpdsFeed struct {
	XMLName xml.Name `xml:"feed"`
	*OpdsCommon
	XmlNs   string       `xml:"xmlns,attr,omitempty"`
	Entries []*OpdsEntry `xml:"entry,omitempty"`
}

type OpdsFeedDB struct {
	*OpdsCommon
	Type    byte
	Sort    byte
	All     bool
	Entries []string
}

type OpdsCommon struct {
	Id      string      `xml:"id,omitempty"`
	Title   string      `xml:"title,omitempty"`
	Links   []*OpdsLink `xml:"link,omitempty"`
	Updated string      `xml:"updated,omitempty"`
	Author  *OpdsAuthor `xml:"author,omitempty"`
}

type OpdsLink struct {
	Rel    string       `xml:"rel,attr,omitempty"`
	Href   string       `xml:"href,attr,omitempty"`
	Type   string       `xml:"type,attr,omitempty"`
	Prices []*OpdsPrice `xml:"http://opds-spec.org/2010/catalog price,omitempty"`
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
	Category string       `xml:"category,omitempty"`
	Content  *OpdsContent `xml:"content,omitempty"`
	Links    []*OpdsLink  `xml:"link,omitempty"`
}

type OpdsMeta struct {
	Title     string      `xml:"title,omitempty"`
	Author    *OpdsAuthor `xml:"author,omitempty"`
	Publisher string      `xml:"http://purl.org/dc/terms/ publisher,omitempty"`
	Issued    string      `xml:"http://purl.org/dc/terms/ issued,omitempty"`
	Lang      string      `xml:"http://purl.org/dc/terms/ language,omitempty"`
	Summary   string      `xml:"summary,omitempty"`
	Rights    string      `xml:"rights,omitempty"`
	Cover     bool        `xml:"-"`
	Thumb     bool        `xml:"-"`
	CoverType string      `xml:"-"`
	ThumbType string      `xml:"-"`
}

type OpdsContent struct {
	Type    string `xml:"type,attr,omitempty"`
	Content string `xml:",chardata"`
}
