package wildcard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWildcard(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(false, Matches("abc", ""))
	assert.Equal(true, Matches("abc", "*"))
	assert.Equal(true, Matches("*", "*"))
	assert.Equal(true, Matches("", "*"))
	assert.Equal(true, Matches("", ""))
	assert.Equal(true, Matches("abc", "a*"))
	assert.Equal(true, Matches("abc", "*b*"))
	assert.Equal(false, Matches("abc", "b*"))
	assert.Equal(false, Matches("abc", "*b"))
}
