package password

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHash_FormatoBcrypt - El hash debe tener el formato bcrypt ($2... con coste por defecto, 60 caracteres)
func TestHash_FormatoBcrypt(t *testing.T) {
	hash, err := Hash("formatoCorrecto123")
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(hash, "$2"), "el hash debe comenzar con el identificador de versión de bcrypt")
	assert.Equal(t, 60, len(hash), "un hash bcrypt con coste por defecto tiene 60 caracteres")
}

// TestVerify_CaseSensitive - La comparación distingue mayúsculas de minúsculas
func TestVerify_CaseSensitive(t *testing.T) {
	hash, err := Hash("ClaveConMayusculas")
	assert.NoError(t, err)
	assert.Error(t, Verify(hash, "claveconmayusculas"))
	assert.NoError(t, Verify(hash, "ClaveConMayusculas"))
}

func TestHash_NotPlaintext(t *testing.T) {
	hash, err := Hash("supersecreto123")
	assert.NoError(t, err)
	assert.NotEqual(t, "supersecreto123", hash)
	assert.NotContains(t, hash, "supersecreto123")
}

func TestHash_DifferentSalts(t *testing.T) {
	h1, err := Hash("mismacreencia")
	assert.NoError(t, err)
	h2, err := Hash("mismacreencia")
	assert.NoError(t, err)
	assert.NotEqual(t, h1, h2, "dos hashes de la misma clave deben diferir por el salt")
}

func TestHash_EmptyPassword(t *testing.T) {
	hash, err := Hash("")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestVerify_CorrectPassword(t *testing.T) {
	hash, err := Hash("miPassword123*")
	assert.NoError(t, err)
	assert.NoError(t, Verify(hash, "miPassword123*"))
}

func TestVerify_WrongPassword(t *testing.T) {
	hash, err := Hash("claveCorrecta")
	assert.NoError(t, err)
	err = Verify(hash, "claveIncorrecta")
	assert.Error(t, err)
}

func TestVerify_EmptyHash(t *testing.T) {
	err := Verify("", "algo")
	assert.Error(t, err)
}

func TestHashAndVerify_TableDriven(t *testing.T) {
	cases := []struct {
		name string
		pw   string
	}{
		{"alfanumerico", "Abc123!@#"},
		{"largo", "estaEsUnaContrsenyaMuyLargaConMuchosCaracteres123"},
		{"unicode", "contrasenaNno123"},
		{"corto", "a1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := Hash(tc.pw)
			assert.NoError(t, err)
			assert.NoError(t, Verify(hash, tc.pw))
			assert.Error(t, Verify(hash, tc.pw+"x"))
		})
	}
}
