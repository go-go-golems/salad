package cmd

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func parseUint32CSV(s string) ([]uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	out := make([]uint32, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "parse channel %q", p)
		}
		out = append(out, uint32(v))
	}
	return out, nil
}
