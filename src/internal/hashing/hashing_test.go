package hashing

import (
	"crypto/sha256"
	"crypto/sha3"
	"crypto/sha512"
	"encoding/hex"
	"testing"
)

func expectedDigests(data []byte) Digests {
	s256 := sha256.Sum256(data)
	s512 := sha512.Sum512(data)
	s3 := sha3.Sum512(data)

	return Digests{
		SHA256:   hex.EncodeToString(s256[:]),
		SHA512:   hex.EncodeToString(s512[:]),
		SHA3_512: hex.EncodeToString(s3[:]),
	}
}

func TestCompute_KnownVectors(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{name: "empty", data: nil},
		{name: "abc", data: []byte("abc")},
		{name: "a", data: []byte("a")},
		{name: "quick", data: []byte("The quick brown fox jumps over the lazy dog")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Compute(tc.data)
			want := expectedDigests(tc.data)

			if got != want {
				t.Fatalf("digests mismatch:\n got: %#v\nwant: %#v", got, want)
			}
			if len(got.SHA256) != 64 {
				t.Fatalf("sha256 hex length: got %d, want 64", len(got.SHA256))
			}
			if len(got.SHA512) != 128 {
				t.Fatalf("sha512 hex length: got %d, want 128", len(got.SHA512))
			}
			if len(got.SHA3_512) != 128 {
				t.Fatalf("sha3-512 hex length: got %d, want 128", len(got.SHA3_512))
			}
		})
	}
}

func TestCompute_VariousSizes(t *testing.T) {
	sizes := []int{0, 1, 2, 3, 31, 32, 33, 255, 256, 1024, 1024 * 1024}
	for _, n := range sizes {
		data := make([]byte, n)
		for i := range data {
			data[i] = byte(i)
		}

		got := Compute(data)
		want := expectedDigests(data)
		if got != want {
			t.Fatalf("size=%d digests mismatch", n)
		}
	}
}
