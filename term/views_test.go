package term

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoundDigits(t *testing.T) {
	assert.Equal(t, 1, countDigits(0))
	assert.Equal(t, 1, countDigits(1))
	assert.Equal(t, 2, countDigits(10))
	assert.Equal(t, 3, countDigits(999))
}
