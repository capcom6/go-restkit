package restkit

import (
	"net/url"
	"testing"
)

func TestSameOrigin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		base string
		ref  string
		want bool
	}{
		{
			name: "empty port matches default",
			base: "http://example.test:/api",
			ref:  "http://example.test/v2",
			want: true,
		},
		{name: "explicit default port", base: "http://EXAMPLE.test:80/api", ref: "http://example.test/v2", want: true},
		{name: "case insensitive host", base: "http://EXAMPLE.test/api", ref: "http://example.test/v2", want: true},
		{name: "different scheme", base: "https://example.test/api", ref: "http://example.test/v2", want: false},
		{name: "different port", base: "http://example.test:8080/api", ref: "http://example.test/v2", want: false},
		{name: "different host", base: "http://example.test/api", ref: "http://other.test/v2", want: false},
		{name: "relative ref", base: "http://example.test/api", ref: "/v2", want: true},
		{name: "protocol relative ref", base: "https://example.test/api", ref: "//example.test/v2", want: true},
		{name: "no base with absolute ref", base: "", ref: "http://example.test/v2", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base, err := url.Parse(tc.base)
			if err != nil {
				t.Fatalf("url.Parse(base) error = %v", err)
			}
			ref, err := url.Parse(tc.ref)
			if err != nil {
				t.Fatalf("url.Parse(ref) error = %v", err)
			}

			if got := sameOrigin(base, ref); got != tc.want {
				t.Errorf("sameOrigin(%q, %q) = %v, want %v", tc.base, tc.ref, got, tc.want)
			}
		})
	}
}
