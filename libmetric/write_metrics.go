package libmetric

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func WriteMetrics(counters []*Counter) error {

	var sb strings.Builder

	for i, c := range counters {

		val := c.value.Load()

		sb.WriteString(c.name)

		if c.labels != "" {
			sb.WriteByte('{')
			sb.WriteString(c.labels)
			sb.WriteByte('}')
		}

		sb.WriteString(" value=")
		sb.WriteString(strconv.FormatUint(val, 10))

		if i < len(counters)-1 {
			sb.WriteByte('\n')
		}
	}

	resp, err := http.Post(
		vmURL+"/write",
		"text/plain",
		strings.NewReader(sb.String()),
	)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("victoriametrics returned %s", resp.Status)
	}

	return nil
}
