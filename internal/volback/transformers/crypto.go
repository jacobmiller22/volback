package transformers

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/jacobmiller22/volume-backup/internal/crypto"
)

type EncryptionTransformer struct {
	Encryptor *crypto.StreamEncryptor
}

func (tf *EncryptionTransformer) Transform(ctx context.Context, input io.Reader, output io.Writer) error {

	// Read all of the data before doing our encryption
	buf, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("error reading all: %v", err)
	}
	r := bytes.NewReader(buf)

	tf.Encryptor.SetReader(r)

	if n, err := io.Copy(output, tf.Encryptor); err != nil {
		return fmt.Errorf("failed to encrypt after reading %d bytes: %v", n, err)
	}
	return nil
}

type DecryptionTransformer struct {
	Decryptor *crypto.StreamDecryptor
}

func (tf *DecryptionTransformer) Transform(ctx context.Context, input io.Reader, output io.Writer) error {
	buf, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("error reading all: %v", err)
	}
	r := bytes.NewReader(buf)

	tf.Decryptor.SetReader(r)

	if n, err := io.Copy(output, tf.Decryptor); err != nil {
		return fmt.Errorf("failed to encrypt after reading %d bytes: %v", n, err)
	}
	return nil
}
