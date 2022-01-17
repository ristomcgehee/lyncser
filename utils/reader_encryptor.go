package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"io/ioutil"
)

type ReaderEncryptor interface {
	EncryptReader(reader io.Reader) (io.Reader, error)
	DecryptReader(reader io.ReadCloser) (io.ReadCloser, error)
}

type AESGCMEncryptor struct {
	Key []byte
}

func (e *AESGCMEncryptor) EncryptReader(reader io.Reader) (io.Reader, error) {
	plaintext, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return nil, fmt.Errorf("error creating cipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("error creating GCM: %w", err)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	readerEncrypted := bytes.NewReader(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("error creating encrypted reader: %w", err)
	}

	return readerEncrypted, nil
}

func (e *AESGCMEncryptor) DecryptReader(reader io.ReadCloser) (io.ReadCloser, error) {
	encryptedData, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading encrypted data: %w", err)
	}

	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return nil, fmt.Errorf("error creating cipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("error creating GCM: %w", err)
	}
	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("error opening GCM: %w", err)
	}
	decryptedReader := io.NopCloser(bytes.NewReader(plaintext))

	return decryptedReader, nil
}
