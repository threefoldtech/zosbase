package crypto

import (
	"crypto/ed25519"
	crypto_rand "crypto/rand"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

type keyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// Generate implements quick.Generator for keyPair
func (keyPair) Generate(rand *rand.Rand, size int) reflect.Value {
	pub, priv, err := ed25519.GenerateKey(crypto_rand.Reader)
	if err != nil {
		panic(err)
	}
	kp := keyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}
	return reflect.ValueOf(kp)
}

// randomMessage generates a random byte slice for testing
type randomMessage []byte

// Generate implements quick.Generator for randomMessage
func (randomMessage) Generate(rand *rand.Rand, size int) reflect.Value {
	// Generate message size between 1 and 1024 bytes
	msgSize := 1 + rand.Intn(1024)
	message := make([]byte, msgSize)
	rand.Read(message)
	return reflect.ValueOf(randomMessage(message))
}

// Property-based tests using testing/quick

func TestEncryptionDecryptionRoundtrip(t *testing.T) {
	f := func(kp keyPair, msg randomMessage) bool {
		message := []byte(msg)
		encrypted, err := Encrypt(message, kp.PublicKey)
		if err != nil {
			return false
		}

		decrypted, err := Decrypt(encrypted, kp.PrivateKey)
		if err != nil {
			return false
		}

		return bytesEqual(decrypted, message)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("encrypt/decrypt roundtrip error message:", err)
	}
}

func TestEncryptionNonDeterminism(t *testing.T) {
	f := func(kp keyPair, msg randomMessage) bool {
		message := []byte(msg)
		encrypted1, err1 := Encrypt(message, kp.PublicKey)
		encrypted2, err2 := Encrypt(message, kp.PublicKey)

		if err1 != nil || err2 != nil {
			return false
		}

		return !bytesEqual(encrypted1, encrypted2)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("encryption is non-deterministic failed with error:", err)
	}
}

func TestEncryptionCiphertextSize(t *testing.T) {
	f := func(kp keyPair, msg randomMessage) bool {
		message := []byte(msg)
		encrypted, err := Encrypt(message, kp.PublicKey)
		if err != nil {
			return false
		}

		// NaCl sealed box adds 48 bytes overhead (32 bytes ephemeral key + 16 bytes MAC)
		expectedSize := len(message) + 48
		return len(encrypted) == expectedSize
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("ciphertext has expected size overhead:", err)
	}
}

func TestECDHEncryptionRoundtrip(t *testing.T) {
	f := func(kp1, kp2 keyPair, msg randomMessage) bool {
		message := []byte(msg)
		encrypted, err := EncryptECDH(message, kp1.PrivateKey, kp2.PublicKey)
		if err != nil {
			return false
		}

		decrypted, err := DecryptECDH(encrypted, kp2.PrivateKey, kp1.PublicKey)
		if err != nil {
			return false
		}

		return bytesEqual(decrypted, message)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("ECDH encrypt/decrypt roundtrip error message:", err)
	}
}

func TestECDHSymmetry(t *testing.T) {
	f := func(kp1, kp2 keyPair, msg randomMessage) bool {
		message := []byte(msg)
		encrypted1, err := EncryptECDH(message, kp1.PrivateKey, kp2.PublicKey)
		if err != nil {
			return false
		}

		decrypted1, err := DecryptECDH(encrypted1, kp2.PrivateKey, kp1.PublicKey)
		if err != nil {
			return false
		}

		encrypted2, err := EncryptECDH(message, kp2.PrivateKey, kp1.PublicKey)
		if err != nil {
			return false
		}

		decrypted2, err := DecryptECDH(encrypted2, kp1.PrivateKey, kp2.PublicKey)
		if err != nil {
			return false
		}

		return bytesEqual(decrypted1, message) &&
			bytesEqual(decrypted2, message) &&
			bytesEqual(decrypted1, decrypted2)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("ECDH encryption is symmetric failed with error:", err)
	}
}

func TestSignatureRoundtrip(t *testing.T) {
	f := func(kp keyPair, msg randomMessage) bool {
		message := []byte(msg)
		signature, err := Sign(kp.PrivateKey, message)
		if err != nil {
			return false
		}

		err = Verify(kp.PublicKey, message, signature)
		return err == nil
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("sign/verify roundtrip succeeds for valid signatures failed with error:", err)
	}
}

func TestSignatureDeterminism(t *testing.T) {
	f := func(kp keyPair, msg randomMessage) bool {
		message := []byte(msg)
		sig1, err1 := Sign(kp.PrivateKey, message)
		sig2, err2 := Sign(kp.PrivateKey, message)

		if err1 != nil || err2 != nil {
			return false
		}

		return bytesEqual(sig1, sig2)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("ed25519 signatures are deterministic failed with error:", err)
	}
}

func TestWrongKeyRejection(t *testing.T) {
	f := func(kp1, kp2 keyPair, msg randomMessage) bool {
		if bytesEqual(kp1.PublicKey, kp2.PublicKey) {
			return true // Skip if keys are the same
		}

		message := []byte(msg)
		signature, err := Sign(kp1.PrivateKey, message)
		if err != nil {
			return false
		}

		err = Verify(kp2.PublicKey, message, signature)
		return err != nil // Should fail with wrong key
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("wrong public key rejects signature failed with error:", err)
	}
}

func TestMessageIntegrity(t *testing.T) {
	f := func(kp keyPair, msg1, msg2 randomMessage) bool {
		message1 := []byte(msg1)
		message2 := []byte(msg2)

		if bytesEqual(message1, message2) {
			return true // Skip if messages are the same
		}

		signature, err := Sign(kp.PrivateKey, message1)
		if err != nil {
			return false
		}

		err = Verify(kp.PublicKey, message2, signature)
		return err != nil // Should fail with tampered message
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("tampered message fails verification failed with error:", err)
	}
}

func TestEncryptionNotAllZeros(t *testing.T) {
	f := func(kp keyPair, msg randomMessage) bool {
		message := []byte(msg)
		encrypted, err := Encrypt(message, kp.PublicKey)
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
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("encryption doesn't produce all-zero output failed with error:", err)
	}
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
