package api

import "testing"

// What people actually paste. Every accepted form here is something someone
// would reasonably type after being told "point a wildcard record at this
// server and enter the domain" — the address bar's version, the DNS record's
// version, and the bare name.
func TestCleanDomainAcceptsWhatPeoplePaste(t *testing.T) {
	for input, want := range map[string]string{
		"example.com":          "example.com",
		"  example.com  ":      "example.com",
		"EXAMPLE.com":          "example.com",
		"https://example.com":  "example.com",
		"http://example.com":   "example.com",
		"https://example.com/": "example.com",
		// Exactly what they were told to create in DNS.
		"*.example.com": "example.com",
		// A trailing dot is a fully qualified name, and correct.
		"example.com.":            "example.com",
		"apps.example.co.uk":      "apps.example.co.uk",
		"my-server.example.com":   "my-server.example.com",
		"xn--80ak6aa92e.example":  "xn--80ak6aa92e.example",
		"1.example.com":           "1.example.com",
		"https://*.example.com/":  "example.com",
		"HTTPS://Example.COM/":    "example.com",
		"deploy.staging.acme.dev": "deploy.staging.acme.dev",
	} {
		got, err := cleanDomain(input)
		if err != nil {
			t.Errorf("cleanDomain(%q) refused it: %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("cleanDomain(%q) = %q, want %q", input, got, want)
		}
	}
}

// Empty is a complete answer, not a mistake: it is how routing is turned off.
func TestCleanDomainTreatsEmptyAsOff(t *testing.T) {
	for _, input := range []string{"", "   "} {
		got, err := cleanDomain(input)
		if err != nil {
			t.Errorf("cleanDomain(%q) should mean 'route nothing', got %v", input, err)
		}
		if got != "" {
			t.Errorf("cleanDomain(%q) = %q, want empty", input, got)
		}
	}
}

// Each of these would be accepted by a laxer check and then fail at Let's
// Encrypt instead, which is a worse place to find out: the certificate request
// is rate limited, so a wrong domain caught late costs a week rather than a
// retype.
func TestCleanDomainRefusesWhatCannotHoldACertificate(t *testing.T) {
	for _, input := range []string{
		"localhost",             // no dot, and not a public name
		"server",                // a hostname, not a domain
		"example.com/dashboard", // a URL with a path
		"example.com:8443",      // a URL with a port
		"two words.com",         // a space
		"exa mple.com",
		"example..com",  // an empty label
		".",             // nothing but separators
		"exam_ple.com",  // underscores are not valid in a hostname label
		"http://server", // scheme stripped, still no dot
	} {
		if got, err := cleanDomain(input); err == nil {
			t.Errorf("cleanDomain(%q) = %q, want it refused", input, got)
		}
	}
}
