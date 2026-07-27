package config

import "testing"

// A mail server that is half-configured is the state worth failing on: it
// looks set up, and it sends nothing.
func TestSMTPConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		from    string
		wantErr bool
		check   func(*testing.T, SMTP)
	}{
		{
			name: "unset is an error, not a quiet no",
			url:  "", from: "", wantErr: true,
		},
		{
			name: "none turns it off",
			url:  "none", from: "",
			check: func(t *testing.T, got SMTP) {
				if got.Enabled() {
					t.Error(`"none" must mean no mail server`)
				}
			},
		},
		{
			name: "a server with no sender is refused",
			url:  "smtp://user:pass@mail.example.com:587", from: "",
			wantErr: true,
		},
		{
			name: "a sender that is not an address is refused",
			url:  "smtp://user:pass@mail.example.com:587", from: "ferrite",
			wantErr: true,
		},
		{
			name: "a port is required, because 25 is rarely what anyone means",
			url:  "smtp://user:pass@mail.example.com", from: "a@b.com",
			wantErr: true,
		},
		{
			name: "an unknown scheme is refused rather than assumed",
			url:  "https://mail.example.com:587", from: "a@b.com",
			wantErr: true,
		},
		{
			name: "starttls",
			url:  "smtp://user:p%40ss@mail.example.com:587", from: "alerts@example.com",
			check: func(t *testing.T, got SMTP) {
				if got.Implicit {
					t.Error("smtp:// starts in the clear and upgrades")
				}
				if got.Host != "mail.example.com" || got.Port != 587 {
					t.Errorf("wrong destination: %s:%d", got.Host, got.Port)
				}
				if got.User != "user" {
					t.Errorf("wrong user: %q", got.User)
				}
				// Escaped in the URL, and a password that arrived still
				// percent-encoded would fail authentication with no clue why.
				if got.Password != "p@ss" {
					t.Errorf("password not decoded: %q", got.Password)
				}
			},
		},
		{
			name: "implicit tls",
			url:  "smtps://user:pass@mail.example.com:465", from: "alerts@example.com",
			check: func(t *testing.T, got SMTP) {
				if !got.Implicit {
					t.Error("smtps:// is TLS from the first byte")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := requireSMTPOrDisabled(test.url, test.from)

			if test.wantErr {
				if err == nil {
					t.Fatalf("want an error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.check != nil {
				test.check(t, got)
			}
		})
	}
}
