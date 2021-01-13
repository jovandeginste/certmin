package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetrieveCerts(t *testing.T) {
	certs, err := retrieveCerts("foo", "faa", true)
	assert.Nil(t, certs)
	assert.Error(t, err)
	if os.Getenv("AUTHOR_TESTING") != "" {
		certs, err = retrieveCerts("tcp", "github.com:443", true)
		assert.NoError(t, err)
		assert.True(t, len(certs) >= 2)
	}
}

func TestParseURL(t *testing.T) {
	remote, err := parseURL("https://foo")
	assert.Equal(t, "foo:443", remote)
	assert.Nil(t, err)

	remote, err = parseURL("ldaps://foo")
	assert.Equal(t, "foo:636", remote)
	assert.Nil(t, err)

	remote, err = parseURL("foo://foo")
	assert.Equal(t, "foo:443", remote)
	assert.Nil(t, err)

	remote, err = parseURL("https://foo:123")
	assert.Equal(t, "foo:123", remote)
	assert.Nil(t, err)

	remote, err = parseURL("foo://foo:123")
	assert.Equal(t, "foo:123", remote)
	assert.Nil(t, err)

	_, err = parseURL("foo://foo:1AA23")
	assert.NotNil(t, err)
}
