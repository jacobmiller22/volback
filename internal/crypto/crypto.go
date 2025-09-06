package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"golang.org/x/crypto/argon2"
)

type StreamDecryptor struct {
	IV       []byte
	S        *cipher.StreamReader
	Password string
}

// GenerateSalt creates a secure random salt of a given length.
func generateSalt(length int) ([]byte, error) {
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

func DeriveEncryptionKey(password string, salt []byte, klen int) ([]byte, error) {
	return argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, uint32(klen)), nil // 1 iteration, 64MB memory, 4 threads, klen-byte key
}

func NewStreamDecryptor(key string) (*StreamDecryptor, error) {
	return &StreamDecryptor{
		IV:       nil,
		S:        nil,
		Password: key,
	}, nil
}

func (c *StreamDecryptor) SetReader(r io.Reader) error {
	c.IV = make([]byte, 16)
	n, err := io.ReadFull(r, c.IV)
	if err != nil {
		return err
	}
	if n != 16 {
		return io.ErrUnexpectedEOF
	}
	salt := make([]byte, 16)
	n, err = io.ReadFull(r, salt)
	if err != nil {
		return err
	}
	if n != 16 {
		return io.ErrUnexpectedEOF
	}

	derivedKey, err := DeriveEncryptionKey(c.Password, salt, 16)
	if err != nil {
		return err
	}

	// Create a block cipher from a secret key
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return err
	}
	c.S = &cipher.StreamReader{
		S: cipher.NewCTR(block, c.IV),
		R: r,
	}

	return nil
}

func (c *StreamDecryptor) Read(p []byte) (n int, err error) {
	return c.S.Read(p)
}

type StreamEncryptor struct {
	Salt   []byte
	IV     []byte
	S      cipher.StreamReader
	offset int // only tracks until the length of IV has been met
}

func NewStreamEncryptor(key string) (*StreamEncryptor, error) {
	salt, err := generateSalt(16)
	if err != nil {
		return nil, err
	}

	derivedKey, err := DeriveEncryptionKey(key, salt, 16)
	if err != nil {
		return nil, err
	}

	// Create a block cipher from a secret key
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	// Create a random nonce
	iv := make([]byte, block.BlockSize())
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	return &StreamEncryptor{
		Salt:   salt,
		IV:     iv,
		S:      cipher.StreamReader{S: cipher.NewCTR(block, iv)},
		offset: 0,
	}, nil
}

func (c *StreamEncryptor) SetReader(r io.Reader) {
	c.S.R = r
}

func (c *StreamEncryptor) Read(p []byte) (n int, err error) {
	ivleft := len(c.IV) - c.offset
	if ivleft > len(p) {
		n := copy(p, c.IV[c.offset:len(p)])
		c.offset += n
		return n, nil
	}
	if ivleft > 0 {
		n := copy(p, c.IV[c.offset:len(c.IV)])
		c.offset += n
		return n, nil
	}
	saltleft := len(c.IV) + len(c.Salt) - c.offset
	if saltleft > len(p) {
		n := copy(p, c.Salt[c.offset-len(c.IV):len(p)])
		c.offset += n
		return n, nil
	}
	if saltleft >= len(c.IV) {
		n := copy(p, c.Salt[c.offset-len(c.IV):len(c.Salt)])
		c.offset += n
		return n, nil
	}
	return c.S.Read(p)
}
