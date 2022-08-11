package main

import (
	"bytes"
	"errors"
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
		server string
	}
	tests := []struct {
		name     string
		args     args
		want     bool
		response func(*http.Request) (*http.Response, error)
	}{
		{
			name: "Error from server",
			args: args{
				query:  "up",
				server: "https://some.server",
			},
			want: false,
			response: func(*http.Request) (*http.Response, error) {
				return nil, errors.New(
					"Error from web server",
				)
			},
		},
		{
			name: "500 Error from server",
			args: args{
				query:  "up",
				server: "https://some.server",
			},
			want: false,
			response: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(bytes.NewReader([]byte(``))),
				}, nil
			},
		},
		{
			name: "200 Empty",
			args: args{
				query:  "up",
				server: "https://some.server",
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
				query:  "up",
				server: "https://some.server",
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
		server string
	}
	tests := []struct {
		name     string
		args     args
		want     Groups
		response func(*http.Request) (*http.Response, error)
	}{
		{
			name: "Error from server",
			args: args{
				server: "https://some.server",
			},
			want: Groups{},
			response: func(*http.Request) (*http.Response, error) {
				return nil, errors.New(
					"Error from web server",
				)
			},
		},
		{
			name: "500 Error from server",
			args: args{
				server: "https://some.server",
			},
			want: Groups{},
			response: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(bytes.NewReader([]byte(``))),
				}, nil
			},
		},
		{
			name: "200 Empty",
			args: args{
				server: "https://some.server",
			},
			want: Groups{},
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
				server: "https://some.server",
			},
			want: Groups{},
			response: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(bytes.NewReader([]byte(`
					{
						"data":{
							"groups": [
								{
									"rules": [{
										"name": "IsUp"
										"query": "up",
										"type": "alerting",
									}]
								}
							]
						}
					}`))),
				}, nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GetDoFunc = tt.response
			if got := getRules(tt.args.server); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRules() = %v, want %v", got, tt.want)
			}
		})
	}
}
