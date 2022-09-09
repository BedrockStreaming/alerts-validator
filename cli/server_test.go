package cli

import (
	"bytes"
	// "errors"
	"io"
	"net/http"
	"reflect"
	"testing"
)

// MockClient is the mock client
type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do is the mock client's `Do` func
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return GetDoFunc(req)
}

var (
	// GetDoFunc fetches the mock client's `Do` func
	GetDoFunc func(req *http.Request) (*http.Response, error)
)

func Test_existingMetric(t *testing.T) {
	type args struct {
		query  string
		server Server
	}
	tests := []struct {
		name     string
		args     args
		want     bool
		response func(*http.Request) (*http.Response, error)
	}{
		{
			name: "200 Empty",
			args: args{
				query: "up",
				server: Server{
					QueryURL: "https://some.server",
				},
			},
			want: false,
			response: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(bytes.NewReader([]byte(`
					{
						"data":{
							"result": []
						}
					}`))),
				}, nil
			},
		},
		{
			name: "200 Not Empty",
			args: args{
				query: "up",
				server: Server{
					QueryURL: "https://some.server",
				},
			},
			want: true,
			response: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(bytes.NewReader([]byte(`
					{
						"data":{
							"result": [
								{
									"metric": "coucou",
									"value": 1
								}
							]
						}
					}`))),
				}, nil
			},
		},
	}
	Client = &MockClient{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GetDoFunc = tt.response
			if got := existingMetric(tt.args.query, tt.args.server); got != tt.want {
				t.Errorf("existingMetric() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getRules(t *testing.T) {
	type args struct {
		server Server
	}
	tests := []struct {
		name     string
		args     args
		want     Response
		response func(*http.Request) (*http.Response, error)
	}{
		{
			name: "200 Empty",
			args: args{
				server: Server{
					RuleURL: "https://some.server",
				},
			},
			want: Response{},
			response: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(bytes.NewReader([]byte(`
					{
					}`))),
				}, nil
			},
		},
		{
			name: "200 Not Empty",
			args: args{
				server: Server{
					RuleURL: "https://some.server",
				},
			},
			want: Response{
				Status: "success",
				Data: Data{
					Groups: []Groups{
						{
							Rules: []Rule{
								{
									Id:         "999999",
									Name:       "AlertmanagerFailedReload",
									Expression: "# Without max_over_time, failed scrapes could create false negatives, see\n# https://www.robustperception.io/alerting-on-gauges-in-prometheus-2-0 for details.\nmax_over_time(alertmanager_config_last_reload_successful{job=\"vmalertmanager-victoria-metrics-k8s-stack\",namespace=\"monitoring\"}[5m]) == 0",
									Type:       "alerting",
								},
							},
						},
					},
				},
			},
			response: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(bytes.NewReader([]byte(`
					  {
						"status": "success",
						"data": {
						  "groups": [
							{
							  "name": "alertmanager.rules",
							  "rules": [
								{
								  "name": "AlertmanagerFailedReload",
								  "query": "# Without max_over_time, failed scrapes could create false negatives, see\n# https://www.robustperception.io/alerting-on-gauges-in-prometheus-2-0 for details.\nmax_over_time(alertmanager_config_last_reload_successful{job=\"vmalertmanager-victoria-metrics-k8s-stack\",namespace=\"monitoring\"}[5m]) == 0",
								  "type": "alerting",
								  "id": "999999"
								}
							  ]
							}
						  ]
						}
					  }
					`))),
				}, nil
			},
		},
	}
	Client = &MockClient{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GetDoFunc = tt.response
			if got := getRules(tt.args.server); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRules() = %v, want %v", got, tt.want)
			}
		})
	}
}
