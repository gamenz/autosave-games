package config

import (
	"path/filepath"
	"strings"
)

// ExpandPath replaces %VAR% tokens and a leading "~" using env, then
// normalizes separators. Unknown variables are kept verbatim.
func ExpandPath(p string, env EnvFunc) string {
	p = expandVars(p, env)
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if home := env("USERPROFILE"); home != "" {
			p = home + p[1:]
		}
	}
	return filepath.Clean(filepath.FromSlash(p))
}

func expandVars(p string, env EnvFunc) string {
	var b strings.Builder
	for {
		start := strings.IndexByte(p, '%')
		if start < 0 {
			break
		}
		end := strings.IndexByte(p[start+1:], '%')
		if end < 0 {
			break
		}
		end += start + 1
		name := p[start+1 : end]
		b.WriteString(p[:start])
		if value := env(name); name != "" && value != "" {
			b.WriteString(value)
		} else {
			b.WriteString(p[start : end+1])
		}
		p = p[end+1:]
	}
	b.WriteString(p)
	return b.String()
}
