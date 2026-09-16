package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
)

type FlowEncryptedRequest struct {
	EncryptedFlowData string `json:"encrypted_flow_data"`
	EncryptedAesKey   string `json:"encrypted_aes_key"`
	InitialVector     string `json:"initial_vector"`
}

type FlowDecryptedRequest struct {
	Version   string         `json:"version"`
	Action    string         `json:"action"` // "INIT", "data_exchange", "ping"
	Screen    string         `json:"screen"`
	Data      map[string]any `json:"data"`
	FlowToken string         `json:"flow_token"`
}

// DecryptFlowPayload decrypts incoming WhatsApp Flow Data Endpoint payloads using RSA-OAEP and AES-128-GCM
func DecryptFlowPayload(req FlowEncryptedRequest, privateKeyPEM string) (*FlowDecryptedRequest, []byte, []byte, error) {
	if privateKeyPEM == "" {
		return nil, nil, nil, fmt.Errorf("WHATSAPP_FLOW_PRIVATE_KEY is not configured")
	}

	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "\\n", "\n")

	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, nil, nil, fmt.Errorf("failed to parse RSA private key PEM block")
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Attempt PKCS8 parsing fallback
		keyUntyped, errPkcs8 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if errPkcs8 != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse private key: PKCS1 (%v), PKCS8 (%v)", err, errPkcs8)
		}
		var ok bool
		privKey, ok = keyUntyped.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, nil, fmt.Errorf("key is not RSA private key")
		}
	}

	// 1. Decrypt AES key using RSA-OAEP SHA-256
	encryptedAesKeyBytes, err := base64.StdEncoding.DecodeString(req.EncryptedAesKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid base64 encrypted_aes_key: %v", err)
	}

	aesKey, err := rsa.DecryptOAEP(sha256.New(), nil, privKey, encryptedAesKeyBytes, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decrypt AES key: %v", err)
	}

	// 2. Decrypt Flow Data using AES-128 GCM
	iv, err := base64.StdEncoding.DecodeString(req.InitialVector)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid base64 initial_vector: %v", err)
	}

	flowDataEncryptedBytes, err := base64.StdEncoding.DecodeString(req.EncryptedFlowData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid base64 encrypted_flow_data: %v", err)
	}

	blockAes, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}

	aesGCM, err := cipher.NewGCMWithNonceSize(blockAes, len(iv))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create GCM AEAD: %v", err)
	}

	decryptedBytes, err := aesGCM.Open(nil, iv, flowDataEncryptedBytes, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decrypt flow payload with GCM: %v", err)
	}

	var decryptedReq FlowDecryptedRequest
	if err := json.Unmarshal(decryptedBytes, &decryptedReq); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal decrypted flow payload: %v", err)
	}

	return &decryptedReq, aesKey, iv, nil
}

// EncryptFlowResponse encrypts outgoing flow response using the AES key and inverted IV per Meta spec
func EncryptFlowResponse(responseObj any, aesKey []byte, iv []byte) (string, error) {
	plainJSON, err := json.Marshal(responseObj)
	if err != nil {
		return "", fmt.Errorf("marshal response: %w", err)
	}

	// Invert IV for response encryption as required by Meta spec
	responseIV := make([]byte, len(iv))
	for i := range iv {
		responseIV[i] = iv[i] ^ 0xFF
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("create AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCMWithNonceSize(block, len(responseIV))
	if err != nil {
		return "", fmt.Errorf("create GCM AEAD: %w", err)
	}

	encryptedBytes := aesGCM.Seal(nil, responseIV, plainJSON, nil)
	return base64.StdEncoding.EncodeToString(encryptedBytes), nil
}

// VerifyWhatsAppWebhookSignature validates Meta X-Hub-Signature-256 HMAC header
func VerifyWhatsAppWebhookSignature(rawBody []byte, signatureHeader string, appSecret string) bool {
	if appSecret == "" || signatureHeader == "" {
		return true // Skip check if secret not configured in dev
	}

	const prefix = "sha256="
	if !strings.HasPrefix(signatureHeader, prefix) {
		return false
	}
	actualSigHex := strings.TrimPrefix(signatureHeader, prefix)

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(rawBody)
	expectedSigBytes := mac.Sum(nil)
	expectedSigHex := hex.EncodeToString(expectedSigBytes)

	return hmac.Equal([]byte(actualSigHex), []byte(expectedSigHex))
}
