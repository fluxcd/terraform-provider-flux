package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestStringList(t *testing.T) {
	origList := []interface{}{"foo", "bar"}
	set := schema.NewSet(schema.HashString, origList)
	list := toStringList(set)

	assert.ElementsMatch(t, origList, list)
}

func TestStringListNil(t *testing.T) {
	list := toStringList(nil)

	assert.Equal(t, list, []string{})
}
