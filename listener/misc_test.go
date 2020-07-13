package listener

import (
	"reflect"
	"testing"
)

func TestToFqdn(t *testing.T) {
	testCases := []struct {
		desc     string
		domain   string
		expected string
	}{
		{
			desc:     "simple",
			domain:   "foo.bar.com",
			expected: "foo.bar.com.",
		},
		{
			desc:     "already FQDN",
			domain:   "foo.bar.com.",
			expected: "foo.bar.com.",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			fqdn := ToFqdn(test.domain)
			if fqdn != test.expected {
				t.Errorf("ToFqdn() = %v, want %v", fqdn, test.expected)
			}
		})
	}
}

func TestUnFqdn(t *testing.T) {
	testCases := []struct {
		desc     string
		fqdn     string
		expected string
	}{
		{
			desc:     "simple",
			fqdn:     "foo.bar.com.",
			expected: "foo.bar.com",
		},
		{
			desc:     "already domain",
			fqdn:     "foo.bar.com",
			expected: "foo.bar.com",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			domain := UnFqdn(test.fqdn)
			if domain != test.expected {
				t.Errorf("UnFqdn() = %v, want %v", domain, test.expected)
			}
		})
	}
}

func Test_getHostNames(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "one host name",
			text: "Host:my-subdomain.example.com",
			want: []string{"my-subdomain.example.com"},
		},
		{
			name: "one host name with white space prefix",
			text: "    Host:my-subdomain.example.com",
			want: []string{"my-subdomain.example.com"},
		},
		{
			name: "one host name with white space suffix",
			text: "Host:my-subdomain.example.com     ",
			want: []string{"my-subdomain.example.com"},
		},
		{
			name: "one host name with white space preffix and suffix",
			text: "  Host:  my-subdomain.example.com     ",
			want: []string{"my-subdomain.example.com"},
		},
		{
			name: "empty hostnames must not be returned",
			text: "  Host:  my-subdomain.example.com,   ,  ",
			want: []string{"my-subdomain.example.com"},
		},
		{
			name: "multiples host names",
			text: "Host:my-subdomain.example.com,www.example.com,example.com",
			want: []string{"my-subdomain.example.com", "www.example.com", "example.com"},
		},
		{
			name: "multiples host names with spaces after comma",
			text: "Host:my-subdomain.example.com, www.example.com, example.com",
			want: []string{"my-subdomain.example.com", "www.example.com", "example.com"},
		},
		{
			name: "host name among attributes",
			text: "Path:/test;Host:my-subdomain.example.com, www.example.com, example.com;Method:GET",
			want: []string{"my-subdomain.example.com", "www.example.com", "example.com"},
		},
		{
			name: "host name among attributes",
			text: "Path:/test;Host:my-subdomain.example.com;Method:GET",
			want: []string{"my-subdomain.example.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHostNamesFromLabel(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHostNamesFromLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}
