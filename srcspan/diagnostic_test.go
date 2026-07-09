package srcspan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	assert.Greater(t, Bug, Error)
	assert.Greater(t, Error, Warning)
	assert.Greater(t, Warning, Note)
	assert.Greater(t, Note, Help)
}
