package crypto

import (
	"crypto/rand"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := "BFBA90B9-7E4E-8E47-BE4A-EAD2C813A040D9183995-0BD0-45BD-8035-53E04598FFB5"

	enc, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == plaintext {
		t.Fatal("ciphertext should differ from plaintext")
	}

	dec, err := Decrypt(key, enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if dec != plaintext {
		t.Fatalf("got %q, want %q", dec, plaintext)
	}
}

func TestEncryptProducesUniqueNonces(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	enc1, _ := Encrypt(key, "same-plaintext")
	enc2, _ := Encrypt(key, "same-plaintext")

	if enc1 == enc2 {
		t.Fatal("two encryptions of the same plaintext should produce different ciphertexts (different nonces)")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)

	enc, _ := Encrypt(key1, "secret")
	_, err := Decrypt(key2, enc)
	if err == nil {
		t.Fatal("decrypting with wrong key should fail")
	}
}
