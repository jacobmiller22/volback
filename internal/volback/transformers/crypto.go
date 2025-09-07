package transformers

import (
	"context"
	"fmt"
	"io"

	"github.com/jacobmiller22/volume-backup/internal/crypto"
)

type EncryptionTransformer struct {
	Encryptor *crypto.StreamEncryptor
}

func (tf *EncryptionTransformer) Transform(ctx context.Context, input io.Reader, output io.Writer) error {
	tf.Encryptor.SetReader(input)

	if n, err := io.Copy(output, tf.Encryptor); err != nil {
		return fmt.Errorf("failed to encrypt after reading %d bytes: %v", n, err)
	}
	return nil
}

type DecryptionTransformer struct {
	Decryptor *crypto.StreamDecryptor
}

func (tf *DecryptionTransformer) Transform(ctx context.Context, input io.Reader, output io.Writer) error {
	tf.Decryptor.SetReader(input)

	if n, err := io.Copy(output, tf.Decryptor); err != nil {
		return fmt.Errorf("failed to encrypt after reading %d bytes: %v", n, err)
	}
	return nil
}
