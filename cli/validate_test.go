package cli

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/rs/zerolog"
)

func Test_checkValidity(t *testing.T) {
	type args struct {
		rule   Rule
		from   string
		to     string
		cache  map[string]bool
		server Server
	}
	tests := []struct {
		name     string
		args     args
		want     map[string]bool
		response func(*http.Request) (*http.Response, error)
	}{
		{
			name: "Alert is invalid",
			args: args{
				rule: Rule{
					Id:         "999999",
					Name:       "AlertmanagerFailedReload",
					Expression: "# Without max_over_time, failed scrapes could create false negatives, see\n# https://www.robustperception.io/alerting-on-gauges-in-prometheus-2-0 for details.\nmax_over_time(alertmanager_config_last_reload_successful{job=\"vmalertmanager-victoria-metrics-k8s-stack\",namespace=\"monitoring\"}[5m]) == 0",
					Type:       "alerting",
				},
				from:  "1h",
				to:    "10h",
				cache: map[string]bool{},
				server: Server{
					QueryURL: "https://some.server",
				},
			},
			want: map[string]bool{
				`alertmanager_config_last_reload_successful{job="vmalertmanager-victoria-metrics-k8s-stack", namespace="monitoring"}`: false,
			},
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
			name: "Alert is valid",
			args: args{
				rule: Rule{
					Id:         "999999",
					Name:       "AlertmanagerFailedReload",
					Expression: "# Without max_over_time, failed scrapes could create false negatives, see\n# https://www.robustperception.io/alerting-on-gauges-in-prometheus-2-0 for details.\nmax_over_time(alertmanager_config_last_reload_successful{job=\"vmalertmanager-victoria-metrics-k8s-stack\",namespace=\"monitoring\"}[5m]) == 0",
					Type:       "alerting",
				},
				from:  "1h",
				to:    "10h",
				cache: map[string]bool{},
				server: Server{
					QueryURL: "https://some.server",
				},
			},
			want: map[string]bool{
				`alertmanager_config_last_reload_successful{job="vmalertmanager-victoria-metrics-k8s-stack", namespace="monitoring"}`: true,
			},
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
		{
			name: "Alert is valid and cached",
			args: args{
				rule: Rule{
					Id:         "999999",
					Name:       "AlertmanagerFailedReload",
					Expression: "# Without max_over_time, failed scrapes could create false negatives, see\n# https://www.robustperception.io/alerting-on-gauges-in-prometheus-2-0 for details.\nmax_over_time(alertmanager_config_last_reload_successful{job=\"vmalertmanager-victoria-metrics-k8s-stack\",namespace=\"monitoring\"}[5m]) == 0",
					Type:       "alerting",
				},
				from: "1h",
				to:   "10h",
				cache: map[string]bool{
					`alertmanager_config_last_reload_successful{job="vmalertmanager-victoria-metrics-k8s-stack", namespace="monitoring"}[1h-10h]`: true,
				},
				server: Server{},
			},
			want: map[string]bool{
				`alertmanager_config_last_reload_successful{job="vmalertmanager-victoria-metrics-k8s-stack", namespace="monitoring"}`: true,
			},
			response: nil,
		},
	}
	Client = &MockClient{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			GetDoFunc = tt.response
			vectors = []string{}
			expr, _ := metricsql.Parse(tt.args.rule.Expression)
			metricsql.VisitAll(expr, tree)

			valids := checkValidity(tt.args.rule, tt.args.from, tt.args.to, tt.args.cache, tt.args.server)
			if !reflect.DeepEqual(valids, tt.want) {
				t.Errorf("checkValidity() = %v, want %v", valids, tt.want)
			}
		})
	}
}
