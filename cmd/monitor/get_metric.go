package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type vmResponse struct {
	Data struct {
		Result []struct {
			Value []any `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func GetMetric(name string, labels ...string) (float64, error) {

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

	query := fmt.Sprintf(`last_over_time(%s[1d])`, sb.String())

	u := fmt.Sprintf(
		"%s/api/v1/query?query=%s",
		envVMURL,
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
