package pkg

import "crypto/sha256"

func Hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return string(bs)
}
