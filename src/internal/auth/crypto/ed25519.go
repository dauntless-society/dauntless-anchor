package crypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
)

var b64 = base64.RawURLEncoding

func DecodePublicKey(publicKeyB64 string) (ed25519.PublicKey, error) {
	pk, err := b64.DecodeString(publicKeyB64)
	if err != nil {
		return nil, err
	}
	if len(pk) != ed25519.PublicKeySize {
		return nil, errors.New("invalid ed25519 public key length")
	}
	return ed25519.PublicKey(pk), nil
}

func DecodeSignature(signatureB64 string) ([]byte, error) {
	sig, err := b64.DecodeString(signatureB64)
	if err != nil {
		return nil, err
	}
	if len(sig) != ed25519.SignatureSize {
		return nil, errors.New("invalid ed25519 signature length")
	}
	return sig, nil
}

func DecodeNonce(nonceB64 string) ([]byte, error) {
	return b64.DecodeString(nonceB64)
}

func VerifySignature(publicKeyB64 string, nonceB64 string, signatureB64 string) error {
	pk, err := DecodePublicKey(publicKeyB64)
	if err != nil {
		return err
	}
	nonce, err := DecodeNonce(nonceB64)
	if err != nil {
		return err
	}
	sig, err := DecodeSignature(signatureB64)
	if err != nil {
		return err
	}
	if !ed25519.Verify(pk, nonce, sig) {
		return errors.New("invalid signature")
	}
	return nil
}
