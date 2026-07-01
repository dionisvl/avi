package slug_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dionisvl/avi/api-go/internal/platform/slug"
)

func TestGenerate(t *testing.T) {
	cases := []struct {
		parts []string
		want  string
	}{
		{[]string{"Бобик", "dog", "Москва"}, "bobik-dog-moskva"},
		{[]string{"Мурка", "cat", "Санкт-Петербург"}, "murka-cat-sankt-peterburg"},
		{[]string{"Buddy", "dog", "moscow"}, "buddy-dog-moscow"},
		{[]string{"  Hello  ", "cat", "city"}, "hello-cat-city"},
		{[]string{"Кот Котович", "cat", "Новосибирск"}, "kot-kotovich-cat-novosibirsk"},
	}
	for _, tc := range cases {
		got := slug.Generate(tc.parts...)
		assert.Equal(t, tc.want, got, "parts=%v", tc.parts)
	}
}

func TestGenerate_MaxLen(t *testing.T) {
	// name=100 chars + city=100 chars would exceed MaxBaseLen without truncation
	longName := strings.Repeat("a", 100)
	longCity := strings.Repeat("b", 100)
	got := slug.Generate(longName, "dog", longCity)
	require.LessOrEqual(t, len(got), slug.MaxBaseLen)
	assert.NotEmpty(t, got)
	// must not end with a dash
	assert.NotEqual(t, '-', got[len(got)-1])
}
