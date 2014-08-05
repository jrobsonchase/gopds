package epub

import "encoding/xml"

type Package struct {
	XMLName  xml.Name    `xml:"package"`
	Meta     Metadata    `xml:"metadata,omitempty"`
	Manifest []Item      `xml:"manifest>item"`
	Guide    []Reference `xml:"guide>reference"`
}

type Item struct {
	Id        string `xml:"id,attr,omitempty"`
	Href      string `xml:"href,attr,omitempty"`
	MediaType string `xml:"media-type,attr,omitempty"`
}

type Metadata struct {
	Title       string `xml:"title"`
	Creator     string `xml:"creator"`
	Publisher   string `xml:"publisher"`
	Format      string `xml:"format"`
	Date        string `xml:"date"`
	Subject     string `xml:"subject"`
	Description string `xml:"description"`
	Rights      string `xml:"rights"`
	Identifier  string `xml:"identifier"`
	Language    string `xml:"language"`
}

type Reference struct {
	Href  string `xm:"href,attr"`
	Title string `xm:"title,attr"`
	Type  string `xm:"type,attr"`
}
