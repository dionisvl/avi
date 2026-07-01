package media

import "testing"

func TestURL(t *testing.T) {
	t.Parallel()

	const base = "http://api.example.com/uploads"

	tests := []struct {
		name      string
		objectKey string
		want      string
	}{
		{"relative key joins base", "items/abc.jpg", base + "/items/abc.jpg"},
		{"absolute https URL passes through", "https://placeholdpicsum.dev/photo/seed/x/800/600", "https://placeholdpicsum.dev/photo/seed/x/800/600"},
		{"absolute http URL passes through", "http://cdn.example.com/y.png", "http://cdn.example.com/y.png"},
		{"empty key joins base", "", base + "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := URL(base, tt.objectKey); got != tt.want {
				t.Fatalf("URL(%q, %q) = %q, want %q", base, tt.objectKey, got, tt.want)
			}
		})
	}
}
