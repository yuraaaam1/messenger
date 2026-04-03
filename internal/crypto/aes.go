package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func Encrypt(key, plaintext []byte) (ciphertext []byte, iv []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	iv = make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}

	ciphertext = aesGCM.Seal(nil, iv, plaintext, nil)
	return ciphertext, iv, nil
}

func Decrypt(key, ciphertext, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesGCM.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
