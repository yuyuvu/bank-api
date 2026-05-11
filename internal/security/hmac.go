package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// ComputeHMAC создаёт подпись по номеру карты, чтобы потом проверить целостность
func ComputeHMAC(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyHMAC сравнивает ожидаемую подпись и только что посчитанную
func VerifyHMAC(data, expectedHMAC string, secret []byte) bool {
	return hmac.Equal([]byte(ComputeHMAC(data, secret)), []byte(expectedHMAC))
}
