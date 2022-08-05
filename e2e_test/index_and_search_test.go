//go:build e2e
// +build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jsphweid/harmondex/cmd"
	"github.com/jsphweid/harmondex/model"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Write code here to run before tests
	cmd.Index(1)
	cmd.LoadServeFiles()

	// Run tests
	exitVal := m.Run()

	os.Exit(exitVal)
}

func createSearchReqBody(notes model.Notes) io.Reader {
	sr := model.SearchRequestBody{Chords: [][]uint8{notes}}
	data, err := json.Marshal(sr)
	if err != nil {
		panic(err.Error())
	}
	return bytes.NewReader(data)
}

func TestBasicCChordE2E(t *testing.T) {
	body := createSearchReqBody([]uint8{60, 64, 67})
	req := httptest.NewRequest(http.MethodPost, "/search", body)
	w := httptest.NewRecorder()
	cmd.HandleSearch(w, req)

	resp := w.Result()
	respBody, _ := io.ReadAll(resp.Body)

	assert := assert.New(t)
	assert.Equal(resp.StatusCode, 200)

	var searchResponse model.SearchResponse
	err := json.Unmarshal(respBody, &searchResponse)
	if err != nil {
		panic(err.Error())
	}

	assert.Equal(searchResponse, model.SearchResponse{
		Start:      0,
		NumMatches: 2,
		NumFiles:   1,
		Results: []model.SearchResultV2{{
			FileId:       1,
			Offsets:      []float32{0, 1},
			MidiMetadata: nil,
		}},
	})
}

func TestBasicFChordE2E(t *testing.T) {
	body := createSearchReqBody([]uint8{60, 65, 69})
	req := httptest.NewRequest(http.MethodPost, "/search", body)
	w := httptest.NewRecorder()
	cmd.HandleSearch(w, req)

	resp := w.Result()
	respBody, _ := io.ReadAll(resp.Body)

	assert := assert.New(t)
	assert.Equal(resp.StatusCode, 200)

	var searchResponse model.SearchResponse
	err := json.Unmarshal(respBody, &searchResponse)
	if err != nil {
		panic(err.Error())
	}

	assert.Equal(searchResponse, model.SearchResponse{
		Start:      0,
		NumMatches: 1,
		NumFiles:   1,
		Results: []model.SearchResultV2{{
			FileId:       1,
			Offsets:      []float32{0.5},
			MidiMetadata: nil,
		}},
	})
}
