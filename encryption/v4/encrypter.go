package encryption

type Encrypter interface {
	Encrypt(plaintext []byte) (Ciphertext, error)
}

type Decrypter interface {
	Decrypt(ciphertext Ciphertext) ([]byte, error)
}

type EncrypterDecrypter interface {
	Encrypter
	Decrypter
}

// Algorithmer is an optional interface that Encrypter implementations can
// satisfy to report which algorithm they use. EncryptMiddleware detects this
// automatically and stores the algorithm in event metadata.
// Third-party Encrypter implementations should implement this interface
// to enable algorithm identification on encrypted events.
type Algorithmer interface {
	Algorithm() Algorithm
}
