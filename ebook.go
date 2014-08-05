package gopds

import "io"

type Ebook interface {
	OpdsMeta() *OpdsMeta
	Cover() io.ReadCloser
	Thumb() io.ReadCloser
	Book() io.ReadCloser
	Close()
}
