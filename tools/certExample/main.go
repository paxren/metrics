package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func readKeys() (*x509.Certificate, *rsa.PrivateKey) {
	// Загружаем сертификат и приватный ключ из файлов ~/cert.pem и ~/private.pem
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	certificateBytes, err := os.ReadFile(filepath.Join(homeDir, "cert.pem"))
	if err != nil {
		log.Fatal(err)
	}

	privateKeyBytes, err := os.ReadFile(filepath.Join(homeDir, "private.pem"))
	if err != nil {
		log.Fatal(err)
	}

	certificatePemBlock, _ := pem.Decode(certificateBytes)
	if certificatePemBlock == nil {
		log.Fatal("certificate not found")
	}

	privateKeyPemBlock, _ := pem.Decode(privateKeyBytes)
	if privateKeyPemBlock == nil {
		log.Fatal("private key not found")
	}

	certificate, err := x509.ParseCertificate(certificatePemBlock.Bytes)
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyPemBlock.Bytes)
	if err != nil {
		log.Fatal(err)
	}

	return certificate, privateKey
}

func main() {
	certificate, privateKey := readKeys()

	// Шифруем сообщение сертификатом (публичным ключом)
	message := []byte("secret message")
	encryptedMessage, err := rsa.EncryptPKCS1v15(rand.Reader, certificate.PublicKey.(*rsa.PublicKey), message)
	if err != nil {
		log.Fatal(err)
	}

	decryptedMessage, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, encryptedMessage)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(bytes.Equal(message, decryptedMessage)) // true
}
