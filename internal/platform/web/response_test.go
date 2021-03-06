package web_test

import (
	"encoding/json"
	"github.com/b2wdigital/restQL-golang/v6/internal/domain"
	"github.com/b2wdigital/restQL-golang/v6/pkg/restql"
	"testing"

	"github.com/b2wdigital/restQL-golang/v6/internal/platform/web"
	"github.com/b2wdigital/restQL-golang/v6/test"
)

func TestMakeQueryResponse(t *testing.T) {
	tests := []struct {
		name        string
		queryResult domain.Resources
		debug       bool
		expected    web.QueryResponse
	}{
		{
			"should make response for simple result",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:       200,
					Success:      true,
					ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  rawResult(`{"id": "12345abcde"}`),
					},
				},
				Headers: map[string]string{},
			},
		},
		{
			"should make response with metadata",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:       200,
					Success:      true,
					IgnoreErrors: true,
					ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true, Metadata: web.StatementMetadata{IgnoreErrors: "ignore"}},
						Result:  rawResult(`{"id": "12345abcde"}`),
					},
				},
				Headers: map[string]string{},
			},
		},
		{
			"should make response with debugging",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:          200,
					Success:         true,
					URL:             "http://hero.io/api",
					RequestHeaders:  map[string]string{"X-Token": "abcabcacbabc"},
					ResponseHeaders: map[string]string{"X-New-Token": "efgefgefg"},
					RequestParams:   map[string]interface{}{"filter": "no"},
					ResponseTime:    100,
					ResponseBody:    restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
				},
			},
			true,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true, Debug: &web.StatementDebugging{
							URL:             "http://hero.io/api",
							RequestHeaders:  map[string]string{"X-Token": "abcabcacbabc"},
							ResponseHeaders: map[string]string{"X-New-Token": "efgefgefg"},
							Params:          map[string]interface{}{"filter": "no"},
							ResponseTime:    100,
						}},
						Result: rawResult(`{"id": "12345abcde"}`),
					},
				},
				Headers: map[string]string{
					"hero-X-New-Token": "efgefgefg",
				},
			},
		},
		{
			"should make response for multiplexed result",
			domain.Resources{
				"hero": restql.DoneResources{
					restql.DoneResource{
						Status:       200,
						Success:      true,
						ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
					},
					restql.DoneResource{
						Status:       200,
						Success:      true,
						ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "67890fghij"}`)),
					},
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: []interface{}{web.StatementDetails{Status: 200, Success: true}, web.StatementDetails{Status: 200, Success: true}},
						Result:  []interface{}{rawResult(`{"id": "12345abcde"}`), rawResult(`{"id": "67890fghij"}`)},
					},
				},
				Headers: map[string]string{},
			},
		},
		{
			"should make response for aggregated multiplexed result",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:       200,
					Success:      true,
					ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "10"}`)),
				},
				"sidekick": restql.DoneResources{
					restql.DoneResource{
						Status:       200,
						Success:      true,
						ResponseBody: &restql.ResponseBody{},
					},
					restql.DoneResource{
						Status:       200,
						Success:      true,
						ResponseBody: &restql.ResponseBody{},
					},
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  rawResult(`{"id": "10"}`),
					},
					"sidekick": {
						Details: []interface{}{web.StatementDetails{Status: 200, Success: true}, web.StatementDetails{Status: 200, Success: true}},
						Result:  nil,
					},
				},
				Headers: map[string]string{},
			},
		},
		{
			"should make response with cache control header for simple result",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:  200,
					Success: true,
					CacheControl: restql.ResourceCacheControl{
						MaxAge:  restql.ResourceCacheControlValue{Exist: true, Time: 400},
						SMaxAge: restql.ResourceCacheControlValue{Exist: true, Time: 300},
					},
					ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  rawResult(`{"id": "12345abcde"}`),
					},
				},
				Headers: map[string]string{"Cache-Control": "max-age=400, s-maxage=300"},
			},
		},
		{
			"should make response with cache control header containing only max-age directive",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:  200,
					Success: true,
					CacheControl: restql.ResourceCacheControl{
						MaxAge: restql.ResourceCacheControlValue{Exist: true, Time: 400},
					},
					ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  rawResult(`{"id": "12345abcde"}`),
					},
				},
				Headers: map[string]string{"Cache-Control": "max-age=400"},
			},
		},
		{
			"should make response with cache control header containing only s-maxage directive",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:  200,
					Success: true,
					CacheControl: restql.ResourceCacheControl{
						SMaxAge: restql.ResourceCacheControlValue{Exist: true, Time: 300},
					},
					ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  rawResult(`{"id": "12345abcde"}`),
					},
				},
				Headers: map[string]string{"Cache-Control": "s-maxage=300"},
			},
		},
		{
			"should make response with cache control header containing only no-cache directive",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:  200,
					Success: true,
					CacheControl: restql.ResourceCacheControl{
						NoCache: true,
					},
					ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  rawResult(`{"id": "12345abcde"}`),
					},
				},
				Headers: map[string]string{"Cache-Control": "no-cache"},
			},
		},
		{
			"should make response with minimum cache control header",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:  200,
					Success: true,
					CacheControl: restql.ResourceCacheControl{
						MaxAge:  restql.ResourceCacheControlValue{Exist: true, Time: 1000},
						SMaxAge: restql.ResourceCacheControlValue{Exist: true, Time: 300},
					},
					ResponseBody: &restql.ResponseBody{},
				},
				"sidekick": restql.DoneResource{
					Status:  200,
					Success: true,
					CacheControl: restql.ResourceCacheControl{
						MaxAge:  restql.ResourceCacheControlValue{Exist: true, Time: 400},
						SMaxAge: restql.ResourceCacheControlValue{Exist: true, Time: 1800},
					},
					ResponseBody: &restql.ResponseBody{},
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  nil,
					},
					"sidekick": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  nil,
					},
				},
				Headers: map[string]string{"Cache-Control": "max-age=400, s-maxage=300"},
			},
		},
		{
			"should make response with minimum cache control header for multiplexed result",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:  200,
					Success: true,
					CacheControl: restql.ResourceCacheControl{
						MaxAge:  restql.ResourceCacheControlValue{Exist: true, Time: 400},
						SMaxAge: restql.ResourceCacheControlValue{Exist: true, Time: 600},
					},
					ResponseBody: &restql.ResponseBody{},
				},
				"sidekick": restql.DoneResources{
					restql.DoneResource{
						Status:  200,
						Success: true,
						CacheControl: restql.ResourceCacheControl{
							MaxAge:  restql.ResourceCacheControlValue{Exist: true, Time: 100},
							SMaxAge: restql.ResourceCacheControlValue{Exist: true, Time: 1800},
						},
						ResponseBody: &restql.ResponseBody{},
					},
					restql.DoneResource{
						Status:  200,
						Success: true,
						CacheControl: restql.ResourceCacheControl{
							MaxAge:  restql.ResourceCacheControlValue{Exist: true, Time: 400},
							SMaxAge: restql.ResourceCacheControlValue{Exist: true, Time: 1800},
						},
						ResponseBody: &restql.ResponseBody{},
					},
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  nil,
					},
					"sidekick": {
						Details: []interface{}{web.StatementDetails{Status: 200, Success: true}, web.StatementDetails{Status: 200, Success: true}},
						Result:  nil,
					},
				},
				Headers: map[string]string{"Cache-Control": "max-age=100, s-maxage=600"},
			},
		},
		{
			"should make response with upstream headers",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:       200,
					Success:      true,
					ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
					ResponseHeaders: map[string]string{
						"TransactionId": "abdcefg",
					},
				},
				"sidekick": restql.DoneResource{
					Status:       200,
					Success:      true,
					ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
					ResponseHeaders: map[string]string{
						"TID": "123456",
					},
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  rawResult(`{"id": "12345abcde"}`),
					},
					"sidekick": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  rawResult(`{"id": "12345abcde"}`),
					},
				},
				Headers: map[string]string{
					"hero-TransactionId": "abdcefg",
					"sidekick-TID":       "123456",
				},
			},
		},
		{
			"should make response with upstream headers except from failed responses",
			domain.Resources{
				"hero": restql.DoneResource{
					Status:       200,
					Success:      true,
					ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
					ResponseHeaders: map[string]string{
						"TransactionId": "abdcefg",
					},
				},
				"sidekick": restql.DoneResource{
					Status:       500,
					Success:      false,
					IgnoreErrors: true,
					ResponseBody: restql.NewResponseBodyFromValue(test.NoOpLogger, test.Unmarshal(`{"id": "12345abcde"}`)),
					ResponseHeaders: map[string]string{
						"TID": "123456",
					},
				},
			},
			false,
			web.QueryResponse{
				StatusCode: 200,
				Body: map[string]web.StatementResult{
					"hero": {
						Details: web.StatementDetails{Status: 200, Success: true},
						Result:  rawResult(`{"id": "12345abcde"}`),
					},
					"sidekick": {
						Details: web.StatementDetails{Status: 500, Success: false, Metadata: web.StatementMetadata{IgnoreErrors: "ignore"}},
						Result:  rawResult(`{"id": "12345abcde"}`),
					},
				},
				Headers: map[string]string{
					"hero-TransactionId": "abdcefg",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := web.MakeQueryResponse(tt.queryResult, tt.debug)
			test.Equal(t, got, tt.expected)
		})
	}

}

func TestCalculateStatusCode(t *testing.T) {
	tests := []struct {
		name        string
		queryResult domain.Resources
		expected    int
	}{
		{
			"should return 200 when resources are successful",
			domain.Resources{
				"hero":     restql.DoneResource{Status: 200},
				"sidekick": restql.DoneResource{Status: 204},
				"villain":  restql.DoneResource{Status: 201},
			},
			200,
		},
		{
			"should return max status code",
			domain.Resources{
				"hero":     restql.DoneResource{Status: 200},
				"sidekick": restql.DoneResource{Status: 500},
				"villain":  restql.DoneResource{Status: 408},
			},
			500,
		},
		{
			"should return max status code",
			domain.Resources{
				"hero": restql.DoneResources{
					restql.DoneResources{
						restql.DoneResource{Status: 200},
						restql.DoneResource{Status: 200},
						restql.DoneResource{Status: 408},
					},
				},
				"sidekick": restql.DoneResource{Status: 204},
				"villain":  restql.DoneResource{Status: 400},
			},
			408,
		},
		{
			"should return max status code expect for result marked with ignore",
			domain.Resources{
				"hero":     restql.DoneResource{Status: 200},
				"sidekick": restql.DoneResource{Status: 500, IgnoreErrors: true},
				"villain":  restql.DoneResource{Status: 400, IgnoreErrors: true},
			},
			200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := web.CalculateStatusCode(tt.queryResult)

			test.Equal(t, got, tt.expected)
		})
	}
}

func rawResult(s string) json.RawMessage {
	b, err := json.Marshal(test.Unmarshal(s))
	if err != nil {
		panic(err)
	}

	return json.RawMessage(b)
}
