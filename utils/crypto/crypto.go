package crypto

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
)

func HmacSha1(content, key, digest string) string {
	pk := []byte(key)
	m := hmac.New(sha1.New, pk)
	m.Write([]byte(content))
	bs := string(m.Sum(nil))
	s := ""
	// %x -> 16进制(hex)
	switch digest {
	case "":
		fallthrough
	case "hex":
		s = fmt.Sprintf("%x", bs)
	default:
		s = fmt.Sprintf("%x", bs)
	}
	return s
}
