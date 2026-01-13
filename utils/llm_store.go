package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LLMProfile stores reusable provider/model/key.
type LLMProfile struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url,omitempty"`
}

const (
	LLMProfileDir  = "output/config"
	LLMProfileFile = "llm_profiles.json.enc"
)

func DefaultLLMProfilePath(baseDir string) string {
	if baseDir == "" {
		baseDir = "output"
	}
	return filepath.Join(baseDir, "config", LLMProfileFile)
}

// LoadLLMProfiles decrypts and loads profiles; requires non-empty secret.
func LoadLLMProfiles(path, secret string) ([]LLMProfile, error) {
	if secret == "" {
		return nil, errors.New("llm secret is empty (set GODSCAN_LLM_SECRET)")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	plain, err := decrypt(b, []byte(secret))
	if err != nil {
		return nil, err
	}
	var out []LLMProfile
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SaveLLMProfiles saves encrypted profiles; requires non-empty secret.
func SaveLLMProfiles(path, secret string, profiles []LLMProfile) error {
	if secret == "" {
		return errors.New("llm secret is empty (set GODSCAN_LLM_SECRET)")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return err
	}
	enc, err := encrypt(raw, []byte(secret))
	if err != nil {
		return err
	}
	return os.WriteFile(path, enc, 0o600)
}

func encrypt(plain, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveKey(key))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := gcm.Seal(nonce, nonce, plain, nil)
	return out, nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveKey(key))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := ciphertext[:gcm.NonceSize()]
	body := ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, body, nil)
}

// deriveKey normalizes arbitrary secret into 32-byte key.
func deriveKey(raw []byte) []byte {
	key := make([]byte, 32)
	for i := 0; i < len(key); i++ {
		key[i] = raw[i%len(raw)]
	}
	return key
}

func UpsertProfile(list []LLMProfile, p LLMProfile) []LLMProfile {
	for i := range list {
		if list[i].Name == p.Name {
			list[i] = p
			return list
		}
	}
	return append(list, p)
}

func DeleteProfile(list []LLMProfile, name string) []LLMProfile {
	var out []LLMProfile
	for _, p := range list {
		if p.Name == name {
			continue
		}
		out = append(out, p)
	}
	return out
}
