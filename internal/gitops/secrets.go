package gitops

import (
	"bytes"
	"fmt"
	"io"

	"filippo.io/age"
	"filippo.io/age/agessh"
)

func decryptSecret(decryptionKey []byte, fileName string, data []byte) ([]byte, error) {
	ident, err := agessh.ParseIdentity(decryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse age identities: %w", err)
	}

	src := bytes.NewReader(data)
	r, err := age.Decrypt(src, ident)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt '%s' age identities: %w", fileName, err)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, r); err != nil {
		return nil, fmt.Errorf("failed to copy age decrypted data into bytes.Buffer: %w", err)
	}

	return b.Bytes(), nil
}
