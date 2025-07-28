package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

type keyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// Generators for property-based testing

// genEd25519KeyPair generates a random ed25519 key pair
func genEd25519KeyPair() gopter.Gen {
	return func(genParams *gopter.GenParameters) *gopter.GenResult {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return &gopter.GenResult{Result: nil, Sieve: nil}
		}
		return gopter.NewGenResult(keyPair{
			PublicKey:  pub,
			PrivateKey: priv,
		}, gopter.NoShrinker)
	}
}

// genMessage generates random byte slices for messages
func genMessage() gopter.Gen {
	return func(genParams *gopter.GenParameters) *gopter.GenResult {
		size := 1 + genParams.Rng.Intn(1024) // 1 to 1024
		result := make([]byte, size)
		for i := range size {
			result[i] = byte(genParams.Rng.Intn(256))
		}
		return gopter.NewGenResult(result, gopter.NoShrinker)
	}
}

// Property-based tests

func TestEncryptionDecryptionRoundtrip(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("encrypt/decrypt roundtrip preserves message", prop.ForAll(
		func(keyPair1 keyPair, message []byte) bool {
			encrypted, err := Encrypt(message, keyPair1.PublicKey)
			if err != nil {
				return false
			}

			decrypted, err := Decrypt(encrypted, keyPair1.PrivateKey)
			if err != nil {
				return false
			}

			return bytesEqual(decrypted, message)
		},
		genEd25519KeyPair(),
		genMessage(),
	))

	properties.TestingRun(t)
}

func TestEncryptionNonDeterminism(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("encryption is non-deterministic", prop.ForAll(
		func(keyPair1 keyPair, message []byte) bool {
			encrypted1, err1 := Encrypt(message, keyPair1.PublicKey)
			encrypted2, err2 := Encrypt(message, keyPair1.PublicKey)

			if err1 != nil || err2 != nil {
				return false
			}

			return !bytesEqual(encrypted1, encrypted2)
		},
		genEd25519KeyPair(),
		genMessage(),
	))

	properties.TestingRun(t)
}

func TestEncryptionCiphertextSize(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("ciphertext has expected size overhead", prop.ForAll(
		func(keyPair1 keyPair, message []byte) bool {
			encrypted, err := Encrypt(message, keyPair1.PublicKey)
			if err != nil {
				return false
			}

			// NaCl sealed box adds 48 bytes overhead (32 bytes ephemeral key + 16 bytes MAC)
			expectedSize := len(message) + 48
			return len(encrypted) == expectedSize
		},
		genEd25519KeyPair(),
		genMessage(),
	))

	properties.TestingRun(t)
}

func TestECDHEncryptionRoundtrip(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("ECDH encrypt/decrypt roundtrip preserves message", prop.ForAll(
		func(keyPair1, keyPair2 keyPair, message []byte) bool {
			encrypted, err := EncryptECDH(message, keyPair1.PrivateKey, keyPair2.PublicKey)
			if err != nil {
				return false
			}

			decrypted, err := DecryptECDH(encrypted, keyPair2.PrivateKey, keyPair1.PublicKey)
			if err != nil {
				return false
			}

			return bytesEqual(decrypted, message)
		},
		genEd25519KeyPair(),
		genEd25519KeyPair(),
		genMessage(),
	))

	properties.TestingRun(t)
}

func TestECDHSymmetry(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("ECDH encryption is symmetric", prop.ForAll(
		func(keyPair1, keyPair2 keyPair, message []byte) bool {
			encrypted1, err := EncryptECDH(message, keyPair1.PrivateKey, keyPair2.PublicKey)
			if err != nil {
				return false
			}

			decrypted1, err := DecryptECDH(encrypted1, keyPair2.PrivateKey, keyPair1.PublicKey)
			if err != nil {
				return false
			}

			encrypted2, err := EncryptECDH(message, keyPair2.PrivateKey, keyPair1.PublicKey)
			if err != nil {
				return false
			}

			decrypted2, err := DecryptECDH(encrypted2, keyPair1.PrivateKey, keyPair2.PublicKey)
			if err != nil {
				return false
			}

			return bytesEqual(decrypted1, message) &&
				bytesEqual(decrypted2, message) &&
				bytesEqual(decrypted1, decrypted2)
		},
		genEd25519KeyPair(),
		genEd25519KeyPair(),
		genMessage(),
	))

	properties.TestingRun(t)
}

func TestSignatureRoundtrip(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("sign/verify roundtrip succeeds for valid signatures", prop.ForAll(
		func(keyPair1 keyPair, message []byte) bool {
			signature, err := Sign(keyPair1.PrivateKey, message)
			if err != nil {
				return false
			}

			err = Verify(keyPair1.PublicKey, message, signature)
			return err == nil
		},
		genEd25519KeyPair(),
		genMessage(),
	))

	properties.TestingRun(t)
}

func TestSignatureDeterminism(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("ed25519 signatures are deterministic", prop.ForAll(
		func(keyPair1 keyPair, message []byte) bool {
			sig1, err1 := Sign(keyPair1.PrivateKey, message)
			sig2, err2 := Sign(keyPair1.PrivateKey, message)

			if err1 != nil || err2 != nil {
				return false
			}

			return bytesEqual(sig1, sig2)
		},
		genEd25519KeyPair(),
		genMessage(),
	))

	properties.TestingRun(t)
}

func TestWrongKeyRejection(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("wrong public key rejects signature", prop.ForAll(
		func(keyPair1, keyPair2 keyPair, message []byte) bool {
			if bytesEqual(keyPair1.PublicKey, keyPair2.PublicKey) {
				return true
			}

			signature, err := Sign(keyPair1.PrivateKey, message)
			if err != nil {
				return false
			}

			err = Verify(keyPair2.PublicKey, message, signature)
			return err != nil
		},
		genEd25519KeyPair(),
		genEd25519KeyPair(),
		genMessage(),
	))

	properties.TestingRun(t)
}

func TestMessageIntegrity(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("tampered message fails verification", prop.ForAll(
		func(keyPair1 keyPair, message1, message2 []byte) bool {
			if bytesEqual(message1, message2) {
				return true
			}

			signature, err := Sign(keyPair1.PrivateKey, message1)
			if err != nil {
				return false
			}

			err = Verify(keyPair1.PublicKey, message2, signature)
			return err != nil
		},
		genEd25519KeyPair(),
		genMessage(),
		genMessage(),
	))

	properties.TestingRun(t)
}

func TestEncryptionNotAllZeros(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("encryption doesn't produce all-zero output", prop.ForAll(
		func(keyPair1 keyPair, message []byte) bool {
			encrypted, err := Encrypt(message, keyPair1.PublicKey)
			if err != nil {
				return false
			}

			allZeros := true
			for _, b := range encrypted {
				if b != 0 {
					allZeros = false
					break
				}
			}

			return !allZeros
		},
		genEd25519KeyPair(),
		genMessage(),
	))

	properties.TestingRun(t)
}

// Helper functions

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
