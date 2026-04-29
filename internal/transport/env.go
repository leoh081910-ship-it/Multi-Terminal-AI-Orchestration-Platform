package transport

import "os"

func mergeCommandEnv(extra map[string]string) []string {
	if len(extra) == 0 {
		return os.Environ()
	}

	env := append([]string{}, os.Environ()...)
	for key, value := range extra {
		env = append(env, key+"="+value)
	}
	return env
}
