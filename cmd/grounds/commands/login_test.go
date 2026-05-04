package commands

import "testing"

func TestLoginSubject(t *testing.T) {
	tests := []struct {
		name      string
		preferred string
		email     string
		want      string
	}{
		{
			name:      "preferred username",
			preferred: "player-one",
			email:     "player@example.com",
			want:      "player-one",
		},
		{
			name:  "email",
			email: "player@example.com",
			want:  "player@example.com",
		},
		{
			name: "current user",
			want: "current user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := loginSubject(tt.preferred, tt.email); got != tt.want {
				t.Fatalf("loginSubject(%q, %q) = %q, want %q", tt.preferred, tt.email, got, tt.want)
			}
		})
	}
}
