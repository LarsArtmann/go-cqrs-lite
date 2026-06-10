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
