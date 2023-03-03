package facts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnv(t *testing.T) {
	input := []string{
		"simple_env_var=value",
		"empty_env_var=",
		"multi_equals=a=b=c",
		"multi_equals_trailing_equals=a=b=c=",
		"multi_equals_no_sep=====",
	}

	expect := map[string]string{
		"simple_env_var":               "value",
		"empty_env_var":                "",
		"multi_equals":                 "a=b=c",
		"multi_equals_trailing_equals": "a=b=c=",
		"multi_equals_no_sep":          "====",
	}

	assert.Equal(t, expect, env(input))
}
