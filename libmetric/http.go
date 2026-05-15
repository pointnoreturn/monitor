package libmetric

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type vmResponse struct {
	Data struct {
		Result []struct {
			Value []any `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func ReadMetric(name string, labels ...string) (float64, error) {

	if len(labels)%2 != 0 {
		return 0, fmt.Errorf("labels must be key/value pairs")
	}

	// build selector
	var sb strings.Builder
	sb.WriteString(name)

	if len(labels) > 0 {
		sb.WriteByte('{')
		for i := 0; i < len(labels); i += 2 {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(labels[i])
			sb.WriteByte('=')
			sb.WriteByte('"')
			sb.WriteString(labels[i+1])
			sb.WriteByte('"')
		}
		sb.WriteByte('}')
	}

	//query := fmt.Sprintf(`last_over_time(%s[1d])`, sb.String())
	query := fmt.Sprintf(`%s`, sb.String())

	u := fmt.Sprintf(
		"%s/api/v1/query?query=%s",
		serviceUrl,
		url.QueryEscape(query),
	)

	resp, err := http.Get(u)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var r vmResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, err
	}

	// no data → start counter at 0
	if len(r.Data.Result) == 0 {
		return 0, nil
	}

	// value format: [timestamp, value]
	valStr := r.Data.Result[0].Value[1].(string)

	var val float64
	fmt.Sscanf(valStr, "%f", &val)

	return val, nil
}
func WriteMetric(name string, value float64, labels ...string) error {
	logger.Debug("[WriteMetric]", "name", name, "labels", labels)

	if len(labels)%2 != 0 {
		return fmt.Errorf("labels must be key/value pairs")
	}

	var sb strings.Builder

	sb.WriteString(name)

	if len(labels) > 0 {
		sb.WriteByte('{')
		for i := 0; i < len(labels); i += 2 {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(labels[i])
			sb.WriteByte('=')
			sb.WriteByte('"')
			sb.WriteString(labels[i+1])
			sb.WriteByte('"')
		}
		sb.WriteByte('}')
	}

	sb.WriteByte(' ')
	sb.WriteString(fmt.Sprintf("%g", value))

	resp, err := http.Post(
		serviceUrl+"/api/v1/import/prometheus",
		"text/plain",
		strings.NewReader(sb.String()),
	)

	logger.Debug(fmt.Sprintf("[WriteMetric] %s", name))

	if err != nil {
		logger.Error(fmt.Sprintf("[WriteMetric] cannot send value %f for %s (%d labels): %v", value, name, len(labels)/2, err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		err := fmt.Errorf("victoriametrics returned %s", resp.Status)
		logger.Debug(fmt.Sprintf("[WriteMetric] cannot send value %f for %s (%d labels): %v", value, name, len(labels)/2, err))
		return err
	}

	logger.Debug(fmt.Sprintf("[WriteMetric] saved value %f for %s (%d labels)", value, name, len(labels)/2))

	return nil
}

func WriteMetrics(counters []*Series) error {

	var sb strings.Builder

	for i, c := range counters {

		val := c.data.Load()

		sb.WriteString(c.name)

		if len(c.labels) > 0 {

			if len(c.labels)%2 != 0 {
				return fmt.Errorf("labels must be key/value pairs")
			}

			sb.WriteByte('{')

			for j := 0; j < len(c.labels); j += 2 {

				if j > 0 {
					sb.WriteByte(',')
				}

				sb.WriteString(c.labels[j])
				sb.WriteByte('=')
				sb.WriteByte('"')
				sb.WriteString(c.labels[j+1])
				sb.WriteByte('"')
			}

			sb.WriteByte('}')
		}

		sb.WriteString(" value=")
		sb.WriteString(strconv.FormatUint(val, 10))

		if i < len(counters)-1 {
			sb.WriteByte('\n')
		}

		logger.Debug(fmt.Sprintf("[WriteMetrics] %s", c.name))
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
