package main

import (
	"fmt"
	"net/http"
	"strings"
)

func WriteMetric(
	name string,
	value float64,
	labels ...string,
) error {

	if len(labels)%2 != 0 {
		return fmt.Errorf("labels must be key/value pairs")
	}

	var sb strings.Builder

	sb.WriteString(name)

	if len(labels) > 0 {
		sb.WriteByte(',')

		for i := 0; i < len(labels); i += 2 {

			if i > 0 {
				sb.WriteByte(',')
			}

			sb.WriteString(labels[i])
			sb.WriteByte('=')
			sb.WriteString(labels[i+1])
		}
	}

	sb.WriteString(" value=")
	sb.WriteString(fmt.Sprintf("%f", value))

	resp, err := http.Post(
		envVMURL+"/write",
		"text/plain",
		strings.NewReader(sb.String()),
	)

	if err != nil {
		appLog.Error(fmt.Sprintf("[WriteMetric] Error for %s, %T %v", name, err, err))
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		err = fmt.Errorf("victoriametrics returned %s", resp.Status)

		appLog.Error(fmt.Sprintf("[WriteMetric] Error for %s, %v", name, err))
		return err
	}

	appLog.Debug(fmt.Sprintf("[WriteMetric] %s = %f", name, value))

	return nil
}
