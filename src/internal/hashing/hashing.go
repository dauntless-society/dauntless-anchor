package hashing

import (
	"crypto/sha256"
	"crypto/sha3"
	"crypto/sha512"
	"encoding/hex"
)

type Digests struct {
	SHA256   string
	SHA512   string
	SHA3_512 string
}

func Compute(data []byte) Digests {
	s256 := sha256.Sum256(data)
	s512 := sha512.Sum512(data)
	s3 := sha3.Sum512(data)

	return Digests{
		SHA256:   hex.EncodeToString(s256[:]),
		SHA512:   hex.EncodeToString(s512[:]),
		SHA3_512: hex.EncodeToString(s3[:]),
	}
}
