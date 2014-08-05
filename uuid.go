package gopds

import (
	"crypto/rand"
	"fmt"
)

func Uuidgen() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}

	b[8] = (b[8] | 0x80) & 0xBF
	b[6] = (b[6] | 0x40) & 0x4F
	return fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
