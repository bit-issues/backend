package users

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// OWASP recommended Argon2id parameters:
// - Memory: 19 MiB (19 * 1024 KiB)
// - Iterations: 2
// - Parallelism: 1
// - Salt length: 16 bytes
// - Key length: 32 bytes.
const (
	argon2idMemory      = 19 * 1024 // 19 MiB in KiB
	argon2idIterations  = 2
	argon2idParallelism = 1
	argon2idSaltLength  = 16
	argon2idKeyLength   = 32
)

func hashPasswordArgon2id(password string) (string, error) {
	salt := make([]byte, argon2idSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2idIterations,
		argon2idMemory,
		argon2idParallelism,
		argon2idKeyLength,
	)

	// Encode as: $argon2id$v=19$m=19456,t=2,p=1$<salt-base64>$<hash-base64>
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2idMemory,
		argon2idIterations,
		argon2idParallelism,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

func verifyPasswordArgon2id(password, encodedHash string) error {
	const valsCount = 6

	// Parse the encoded hash: $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
	vals := strings.Split(encodedHash, "$")
	if len(vals) != valsCount {
		return fmt.Errorf("%w: invalid hash", ErrInvalidCredential)
	}
	if vals[1] != "argon2id" || vals[2] != "v="+strconv.Itoa(argon2.Version) {
		return fmt.Errorf("%w: invalid hash", ErrInvalidCredential)
	}

	// vals[0] - пустая строка
	// vals[1] - "argon2id"
	// vals[2] - "v=19"
	// vals[3] - "m=19456,t=2,p=1"

	var memory, iterations, parallelism uint32
	_, err := fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return fmt.Errorf("%w: invalid hash", ErrInvalidCredential)
	}

	salt, err := base64.StdEncoding.DecodeString(vals[4])
	if err != nil {
		return fmt.Errorf("%w: invalid hash", ErrInvalidCredential)
	}

	hash, err := base64.StdEncoding.DecodeString(vals[5])
	if err != nil {
		return fmt.Errorf("%w: invalid hash", ErrInvalidCredential)
	}

	if parallelism == 0 || parallelism > math.MaxUint8 || len(hash) > math.MaxUint16 {
		return fmt.Errorf("%w: invalid hash", ErrInvalidCredential)
	}

	// Compute hash of the provided password with same parameters
	computedHash := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		uint8(parallelism),
		uint32(len(hash)), //nolint:gosec // checked abpve
	)

	// Constant-time comparison
	if subtle.ConstantTimeCompare(hash, computedHash) == 0 {
		return ErrInvalidCredential
	}

	return nil
}
