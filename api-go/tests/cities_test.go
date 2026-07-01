package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cityapi "github.com/dionisvl/avi/api-go/internal/api/city"
)

func TestCitiesList(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/cities", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp cityapi.ListResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Data, 15)

	bySlug := make(map[string]int, len(resp.Data))
	for i, city := range resp.Data {
		bySlug[city.Slug] = i
		assert.NotEmpty(t, city.ID)
		assert.NotEmpty(t, city.Names["ru"])
		assert.NotEmpty(t, city.Names["en"])
		assert.True(t, city.IsActive)
		assert.Greater(t, city.Population, 0)
	}

	newYork := resp.Data[bySlug["new-york"]]
	require.NotNil(t, newYork.GeonameID)
	assert.Equal(t, 5128581, *newYork.GeonameID)

	_, hasLondon := bySlug["london"]
	assert.True(t, hasLondon)

	// cities must be sorted by population descending
	for i := 1; i < len(resp.Data); i++ {
		assert.GreaterOrEqual(t, resp.Data[i-1].Population, resp.Data[i].Population)
	}
}
