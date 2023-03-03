package facts

import (
	"os"
	"strings"
)

func Env() map[string]string {
	return env(os.Environ())
}

func env(envs []string) map[string]string {
	ret := map[string]string{}
	for _, env := range envs {
		split := strings.SplitN(env, "=", 2)
		ret[split[0]] = split[1]
	}
	return ret
}
