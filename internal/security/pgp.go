package security

import (
	"fmt"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/ProtonMail/gopenpgp/v3/profile"
)

// PGPService шифрует и расшифровывает поля карт
type PGPService struct {
	publicKey  string
	privateKey string
	passphrase string
}

// NewPGPService создаёт сервис для шифрования данных карт; при отсутствии заданных ключей генерирует новую пару
func NewPGPService(pubKey, privKey, passphrase string) (*PGPService, error) {
	if pubKey == "" || privKey == "" {
		genPub, genPriv, err := generateKeyPair(passphrase)
		if err != nil {
			return nil, fmt.Errorf("генерация ключей: %w", err)
		}
		pubKey = genPub
		privKey = genPriv
	}
	return &PGPService{
		publicKey:  pubKey,
		privateKey: privKey,
		passphrase: passphrase,
	}, nil
}

// PublicKeyArmored возвращает публичный ключ в armored-виде, чтобы его можно было сохранить между рестартами приложения
func (s *PGPService) PublicKeyArmored() string {
	return s.publicKey
}

// PrivateKeyArmored возвращает приватный ключ в armored-виде, чтобы его можно было сохранить между рестартами приложения
func (s *PGPService) PrivateKeyArmored() string {
	return s.privateKey
}

// generateKeyPair генерирует PGP-ключи по стандарту RFC 9580
func generateKeyPair(passphrase string) (string, string, error) {
	pgp := crypto.PGPWithProfile(profile.RFC9580())

	// Генерация ключа без пароля
	key, err := pgp.KeyGeneration().
		AddUserId("bank", "bank@example.com").
		New().
		GenerateKey()
	if err != nil {
		return "", "", fmt.Errorf("сбой генерации ключа: %w", err)
	}

	// Шифрование приватного ключа паролем
	lockedKey, err := pgp.LockKey(key, []byte(passphrase))
	if err != nil {
		return "", "", fmt.Errorf("сбой шифрования ключа: %w", err)
	}

	// Экспорт публичного ключа
	pubKey, err := key.GetArmoredPublicKey()
	if err != nil {
		return "", "", err
	}

	// Экспорт зашифрованного приватного ключа
	privKey, err := lockedKey.Armor()
	if err != nil {
		return "", "", err
	}

	return pubKey, privKey, nil
}

// Encrypt шифрует строку публичным ключом
func (s *PGPService) Encrypt(plaintext string) (string, error) {
	pgp := crypto.PGPWithProfile(profile.RFC9580())

	pubKey, err := crypto.NewKeyFromArmored(s.publicKey)
	if err != nil {
		return "", fmt.Errorf("импорт публичного ключа: %w", err)
	}

	encHandle, err := pgp.Encryption().Recipient(pubKey).New()
	if err != nil {
		return "", fmt.Errorf("создание шифратора: %w", err)
	}

	pgpMessage, err := encHandle.Encrypt([]byte(plaintext))
	if err != nil {
		return "", fmt.Errorf("шифрование: %w", err)
	}

	armored, err := pgpMessage.ArmorBytes()
	if err != nil {
		return "", fmt.Errorf("армирование: %w", err)
	}

	return string(armored), nil
}

// Decrypt расшифровывает строку приватным ключом
func (s *PGPService) Decrypt(armoredData string) (string, error) {
	pgp := crypto.PGPWithProfile(profile.RFC9580())

	// Попытка импортировать как зашифрованный, так и открытый ключ
	privKey, err := crypto.NewPrivateKeyFromArmored(s.privateKey, []byte(s.passphrase))
	if err != nil {
		return "", fmt.Errorf("импорт приватного ключа: %w", err)
	}

	decHandle, err := pgp.Decryption().DecryptionKey(privKey).New()
	if err != nil {
		return "", fmt.Errorf("создание дешифратора: %w", err)
	}

	decrypted, err := decHandle.Decrypt([]byte(armoredData), crypto.Armor)
	if err != nil {
		return "", fmt.Errorf("дешифрование: %w", err)
	}

	return string(decrypted.Bytes()), nil
}
