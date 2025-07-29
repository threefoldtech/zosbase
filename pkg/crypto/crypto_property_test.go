package crypto

import (
	"bytes"
	"crypto/ed25519"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

const (
	// NaClSealedBoxOverhead represents the overhead added by NaCl sealed box encryption
	// (32 bytes ephemeral key + 16 bytes MAC)
	NaClSealedBoxOverhead = 48
)

type keyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// Generate implements quick.Generator for keyPair
func (keyPair) Generate(rand *rand.Rand, size int) reflect.Value {
	pub, priv, err := ed25519.GenerateKey(rand)
	if err != nil {
		// Return zero value if key generation fails
		// This will cause the test to fail gracefully rather than panic
		return reflect.ValueOf(keyPair{})
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
func encryptThenDecrypt(message []byte, publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey) ([]byte, []byte, error) {
	encrypted, err := Encrypt(message, publicKey)
	if err != nil {
		return nil, nil, err
	}

	decrypted, err := Decrypt(encrypted, privateKey)
	if err != nil {
		return encrypted, nil, err
	}

	return encrypted, decrypted, nil
}

func encryptThenDecryptECDH(message []byte, privateKey1 ed25519.PrivateKey, publicKey1 ed25519.PublicKey, privateKey2 ed25519.PrivateKey, publicKey2 ed25519.PublicKey) ([]byte, []byte, error) {
	encrypted, err := EncryptECDH(message, privateKey1, publicKey2)
	if err != nil {
		return nil, nil, err
	}

	decrypted, err := DecryptECDH(encrypted, privateKey2, publicKey1)
	if err != nil {
		return encrypted, nil, err
	}

	return encrypted, decrypted, nil
}

// isValidKeyPair checks if a keyPair has valid non-zero keys
func isValidKeyPair(kp keyPair) bool {
	return len(kp.PublicKey) > 0 && len(kp.PrivateKey) > 0
}

// Property-based tests using testing/quick

func TestEncryptionDecryptionRoundtrip(t *testing.T) {
	f := func(kp keyPair, msg randomMessage) bool {
		if !isValidKeyPair(kp) {
			return true
		}

		message := []byte(msg)
		_, decrypted, err := encryptThenDecrypt(message, kp.PublicKey, kp.PrivateKey)
		if err != nil {
			return false
		}
		return bytes.Equal(decrypted, message)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("encrypt/decrypt roundtrip error message:", err)
	}
}

func TestEncryptionNonDeterminism(t *testing.T) {
	f := func(kp keyPair, msg randomMessage) bool {
		if !isValidKeyPair(kp) {
			return true
		}

		message := []byte(msg)
		encrypted1, _, err1 := encryptThenDecrypt(message, kp.PublicKey, kp.PrivateKey)
		encrypted2, _, err2 := encryptThenDecrypt(message, kp.PublicKey, kp.PrivateKey)

		if err1 != nil || err2 != nil {
			return false
		}

		return !bytes.Equal(encrypted1, encrypted2)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("encryption is non-deterministic failed with error:", err)
	}
}

func TestEncryptionCiphertextSize(t *testing.T) {
	f := func(kp keyPair, msg randomMessage) bool {
		if !isValidKeyPair(kp) {
			return true
		}

		message := []byte(msg)
		encrypted, _, err := encryptThenDecrypt(message, kp.PublicKey, kp.PrivateKey)
		if err != nil {
			return false
		}

		expectedSize := len(message) + NaClSealedBoxOverhead
		return len(encrypted) == expectedSize
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("ciphertext has expected size overhead:", err)
	}
}

func TestECDHEncryptionRoundtrip(t *testing.T) {
	f := func(kp1, kp2 keyPair, msg randomMessage) bool {
		if !isValidKeyPair(kp1) || !isValidKeyPair(kp2) {
			return true
		}

		message := []byte(msg)
		_, decrypted, err := encryptThenDecryptECDH(message, kp1.PrivateKey, kp1.PublicKey, kp2.PrivateKey, kp2.PublicKey)
		if err != nil {
			return false
		}
		return bytes.Equal(decrypted, message)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("ECDH encrypt/decrypt roundtrip error message:", err)
	}
}

func TestECDHSymmetry(t *testing.T) {
	f := func(kp1, kp2 keyPair, msg randomMessage) bool {
		if !isValidKeyPair(kp1) || !isValidKeyPair(kp2) {
			return true
		}

		message := []byte(msg)
		_, decrypted1, err := encryptThenDecryptECDH(message, kp1.PrivateKey, kp1.PublicKey, kp2.PrivateKey, kp2.PublicKey)
		if err != nil {
			return false
		}

		_, decrypted2, err := encryptThenDecryptECDH(message, kp2.PrivateKey, kp2.PublicKey, kp1.PrivateKey, kp1.PublicKey)
		if err != nil {
			return false
		}

		return bytes.Equal(decrypted1, message) &&
			bytes.Equal(decrypted2, message) &&
			bytes.Equal(decrypted1, decrypted2)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("ECDH encryption is symmetric failed with error:", err)
	}
}

func TestSignatureRoundtrip(t *testing.T) {
	f := func(kp keyPair, msg randomMessage) bool {
		if !isValidKeyPair(kp) {
			return true
		}

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
		if !isValidKeyPair(kp) {
			return true
		}

		message := []byte(msg)
		sig1, err1 := Sign(kp.PrivateKey, message)
		sig2, err2 := Sign(kp.PrivateKey, message)

		if err1 != nil || err2 != nil {
			return false
		}

		return bytes.Equal(sig1, sig2)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("ed25519 signatures are deterministic failed with error:", err)
	}
}

func TestWrongKeyRejection(t *testing.T) {
	f := func(kp1, kp2 keyPair, msg randomMessage) bool {
		if !isValidKeyPair(kp1) || !isValidKeyPair(kp2) {
			return true
		}

		if bytes.Equal(kp1.PublicKey, kp2.PublicKey) {
			return true // Skip if keys are the same
		}

		message := []byte(msg)
		signature, err := Sign(kp1.PrivateKey, message)
		if err != nil {
			return false
		}

		err = Verify(kp2.PublicKey, message, signature)
		return err != nil
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("wrong public key rejects signature failed with error:", err)
	}
}

func TestMessageIntegrity(t *testing.T) {
	f := func(kp keyPair, msg1, msg2 randomMessage) bool {
		if !isValidKeyPair(kp) {
			return true
		}

		message1 := []byte(msg1)
		message2 := []byte(msg2)

		if bytes.Equal(message1, message2) {
			return true // Skip if messages are the same
		}

		signature, err := Sign(kp.PrivateKey, message1)
		if err != nil {
			return false
		}

		err = Verify(kp.PublicKey, message2, signature)
		return err != nil
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error("tampered message fails verification failed with error:", err)
	}
}

func TestEncryptionNotAllZeros(t *testing.T) {
	f := func(kp keyPair, msg randomMessage) bool {
		if !isValidKeyPair(kp) {
			return true
		}

		message := []byte(msg)
		encrypted, _, err := encryptThenDecrypt(message, kp.PublicKey, kp.PrivateKey)
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
