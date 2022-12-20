package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func concatImageAndTag(img, tag string) string {
	return img + ":" + tag
}

func TestImageAndTagFromPaths(t *testing.T) {
	assert := assert.New(t)

	paths := make([]string, 0)

	assert.Equal(concatImageAndTag("*", ""), concatImageAndTag(getImageAndTag(paths)))

	paths = make([]string, 1)

	paths[0] = "*"
	assert.Equal(concatImageAndTag("*", ""), concatImageAndTag(getImageAndTag(paths)))
	paths[0] = ":"
	assert.Equal(concatImageAndTag("*", "*"), concatImageAndTag(getImageAndTag(paths)))
	paths[0] = "*:"
	assert.Equal(concatImageAndTag("*", "*"), concatImageAndTag(getImageAndTag(paths)))
	paths[0] = ":*"
	assert.Equal(concatImageAndTag("*", "*"), concatImageAndTag(getImageAndTag(paths)))
	paths[0] = "*abc*:*abc*"
	assert.Equal(concatImageAndTag("*abc*", "*abc*"), concatImageAndTag(getImageAndTag(paths)))
}
