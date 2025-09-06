package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDeriveEncryptionKeyIdempotent(t *testing.T) {

	pass := "test password"
	salt := []byte{234, 240, 121, 35, 3, 0}

	key1, err1 := DeriveEncryptionKey(pass, salt, 16)

	key2, err2 := DeriveEncryptionKey(pass, salt, 16)

	if err1 != err2 {
		t.Errorf("Mismatch in expected values of err. Expected %+v and %+v to match", err1, err2)
	}

	if !reflect.DeepEqual(key1, key2) {
		t.Errorf("Mismatch in expected values of key. Expected %+v and %+v to match", key1, key2)
	}

}

func TestDeriveEncryptionKey(t *testing.T) {

	pass := "test password"
	// salt := []byte{0, 1, 2, 154, 234, 240, 121, 35, 3, 0, 156, 82, 32, 9, 32, 65}
	salt := []byte{234, 240, 121, 35, 3, 0}

	key, err := DeriveEncryptionKey(pass, salt, 16)

	if err != nil {
		t.Fatalf("error creating encryption key: %+v", err)
	}

	if len(key) != 16 {
		t.Errorf("Mismatch in key length.\n-want: %d\n+got: %d", 16, len(key))
	}

	testCases := []struct {
		klen int
		pass string
		salt []byte
	}{
		{
			klen: 16,
			pass: "test",
			salt: []byte{234, 240, 121, 35, 3, 0},
		},
		{
			klen: 8,
			pass: "test",
			salt: []byte{234, 240, 121, 35, 3, 0},
		},
		{
			klen: 1,
			pass: "test",
			salt: []byte{234, 240, 121, 35, 3, 0},
		},
		{
			klen: 1,
			pass: "test",
			salt: nil,
		},
		{
			klen: 1,
			pass: "testkfjdskjfhsdlkghsdjlghsd;jghsdjkg;hsdjghsdjkghsdkghsd",
			salt: nil,
		},
	}

	for _, tc := range testCases {
		key, err := DeriveEncryptionKey(tc.pass, tc.salt, tc.klen)

		if err != nil {
			t.Fatalf("error creating encryption key: %+v", err)
		}

		if len(key) != tc.klen {
			t.Errorf("Mismatch in key length.\n-want: %d\n+got: %d", tc.klen, len(key))
		}
	}
}

func TestNewStreamDecryptor(t *testing.T) {

	pass := "test key"

	decryptor, err := NewStreamDecryptor(pass)

	if err != nil {
		t.Fatalf("error creating stream decryptor: %+v", err)
	}

	if decryptor == nil {
		t.Error("Mismatch in decryptor value.\n-want: not nil\n+got: nil")
	}
}

// func TestEncryptDecrypt(t *testing.T) {

// 	input := strings.NewReader("this is some test data that is longer than a single block size")

// 	pass := "test key"

// 	encryptor, err := NewStreamEncryptor(pass)
// 	if err != nil {
// 		t.Fatalf("error creating stream encryptor: %+v", err)
// 	}

// 	if encryptor == nil {
// 		t.Error("Mismatch in encryptor value.\n-want: not nil\n+got: nil")
// 	}

// 	decryptor, err := NewStreamDecryptor(pass)
// 	if err != nil {
// 		t.Fatalf("error creating stream decryptor: %+v", err)
// 	}

// 	if decryptor == nil {
// 		t.Error("Mismatch in decryptor value.\n-want: not nil\n+got: nil")
// 	}
// 	encryptor.SetReader(input)
// 	decryptor.SetReader(encryptor)

// 	output, err := io.ReadAll(decryptor)
// 	if err != nil {
// 		t.Fatalf("error reading decrypted output: %+v", err)
// 	}

// 	expected := "this is some test data that is longer than a single block size"
// 	if string(output) != expected {
// 		t.Errorf("Mismatch in decrypted output.\n-want: %q\n+got: %q", expected, output)
// 	}
// }

func TestEncrypt(t *testing.T) {

	key := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("error creating block cipher: %+v", err)
	}

	iv := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	salt := []byte{16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}

	r := strings.NewReader("this is some test data that is longer than a single block size")
	enc := StreamEncryptor{
		Salt: salt,
		IV:   iv,
		S:    cipher.StreamReader{S: cipher.NewCTR(block, iv)},
	}

	enc.SetReader(r)

	output, err := io.ReadAll(&enc)
	if err != nil {
		t.Fatalf("error reading encrypted output: %+v", err)
	}

	// Expect the output to first have the IV, then the salt, then the encrypted data
	if len(output) < len(iv)+len(salt) {
		t.Fatalf("encrypted output is too short: %d", len(output))
	}

	if diff := cmp.Diff(iv, output[0:len(iv)]); diff != "" {
		t.Errorf("IV mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(salt, output[len(iv):len(iv)+len(salt)]); diff != "" {
		t.Errorf("Salt mismatch (-want +got):\n%s", diff)
	}
}
