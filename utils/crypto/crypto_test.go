package crypto

import (
	"sort"
	"strings"
	"testing"
)

var (
	originalObj = map[string]string{
		"timeLine":  "1515750862",
		"path":      "/oauth/v2/token",
		"grantType": "clientcredentials",
		"userName":  "70031400",
		"passWord":  "123456",
		"clientId":  "20000",
		"brandId":   "524726977",
	}
	originaText = `brandId=524726977&clientId=20000&grantType=clientcredentials&passWord=123456&path=/oauth/v2/token&timeLine=1515750862&userName=70031400`
	key         = "0cd684826127ecf963b87b6939cfe947"
	target      = "40555bcf3f65bb35a20d52bc677895808fab726c"
)

func TestHmacSha1(t *testing.T) {
	text := generateOriginalText4Sign(originalObj)
	res := HmacSha1(text, key, "hex")
	if res != target {
		t.Errorf(`HmacSha1(%v, %v, "hex") = (%v), want(%v)`, text, key, res, target)
	}
}

func TestGenerateOriginalText4Sign(t *testing.T) {
	text := generateOriginalText4Sign(originalObj)
	if text != originaText {
		t.Errorf(`generateOriginalText4Sign(originalObj) = (%v), want(%v)`, text, originaText)
	}
}

func generateOriginalText4Sign(originalObj map[string]string) string {
	keys := []string{}
	vals := []string{}
	for k, _ := range originalObj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, v := range keys {
		vals = append(vals, v+"="+originalObj[v])
	}
	text := strings.Join(vals, "&")
	return text
}
