package priv_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/acoustid/go-acoustid/chromaprint"
	"github.com/acoustid/priv"
	"github.com/acoustid/priv/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFingerprint = "AQAAZFKYSFKYofGJj0IOUTRy_AgTch1axYidILR0mFENmcdxfEiL9jiuH8089EJ7-B3yQexzVFWOboeI60h_HHWMHiZ3hCwLXTzy4JTxRsfX4cqI45IpInTCIL1x9EZEbcd7tJVhDfrxwzt8HD3-D9p2XDq0D0cY0agV_EKL78dPPBeC7byQv0IdHUdzdD_wO8g5QeOPtBX66EFn2Jpx5Ucz_Th2ovkMPrgaycgOGVtjI19x_DiR_gAAyHFGUJGgUAAw4JQBQDAHCUIKEIKQEBA4gpFyyEiEACSgEQCIMVYyIwBQwiiBBDFIG0QIEY4AQAAAGgkEnHFXaCQA"

func makeRequest(t *testing.T, s *priv.API, method string, path string, body io.Reader) (int, string) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, path, body)
	require.NoError(t, err)
	s.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

func TestApi_Health(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mock.NewMockService(ctrl)
	service.EXPECT().Status().AnyTimes().Return(true)

	api := priv.NewAPI(service)

	{
		status, body := makeRequest(t, api, "GET", "/_health", nil)
		assert.Equal(t, http.StatusOK, status)
		assert.JSONEq(t, `{}`, body)
	}

	api.SetHealthStatus(false)

	{
		status, body := makeRequest(t, api, "GET", "/_health", nil)
		assert.Equal(t, http.StatusServiceUnavailable, status)
		assert.JSONEq(t, `{"status":503,"error":{"type":"unavailable","reason":"Service is unavailable"}}`, body)
	}

	api.SetHealthStatus(true)

	{
		status, body := makeRequest(t, api, "GET", "/_health", nil)
		assert.Equal(t, http.StatusOK, status)
		assert.JSONEq(t, `{}`, body)
	}
}

func createMockCatalogService(ctrl *gomock.Controller) (*mock.MockService, *mock.MockCatalog) {
	catalog := mock.NewMockCatalog(ctrl)
	catalog.EXPECT().Name().AnyTimes().Return("cat1")

	repo := mock.NewMockRepository(ctrl)
	repo.EXPECT().Catalog("cat1").Return(catalog)

	account := mock.NewMockAccount(ctrl)
	account.EXPECT().Repository().Return(repo)

	service := mock.NewMockService(ctrl)
	service.EXPECT().GetAccount(gomock.Any()).Return(account, nil)

	return service, catalog
}

func TestApi_ListCatalogs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	catalog1 := mock.NewMockCatalog(ctrl)
	catalog1.EXPECT().Name().AnyTimes().Return("cat1")

	catalog2 := mock.NewMockCatalog(ctrl)
	catalog2.EXPECT().Name().AnyTimes().Return("cat2")

	repo := mock.NewMockRepository(ctrl)
	repo.EXPECT().ListCatalogs().Return([]priv.Catalog{catalog1, catalog2}, nil)

	account := mock.NewMockAccount(ctrl)
	account.EXPECT().Repository().Return(repo)

	service := mock.NewMockService(ctrl)
	service.EXPECT().GetAccount(gomock.Any()).Return(account, nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "GET", "/v1/priv", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalogs": [{"catalog": "cat1"}, {"catalog": "cat2"}]}`, body)
}

func TestApi_ListCatalogs_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	repo.EXPECT().ListCatalogs().Return([]priv.Catalog{}, nil)

	account := mock.NewMockAccount(ctrl)
	account.EXPECT().Repository().Return(repo)

	service := mock.NewMockService(ctrl)
	service.EXPECT().GetAccount(gomock.Any()).Return(account, nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "GET", "/v1/priv", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalogs": []}`, body)
}

func TestApi_DeleteCatalog(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().DeleteCatalog().Return(nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "DELETE", "/v1/priv/cat1", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog": "cat1"}`, body)
}

func TestApi_DeleteCatalog_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().DeleteCatalog().Return(errors.New("failed"))

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "DELETE", "/v1/priv/cat1", nil)
	assertHTTPInternalError(t, status, body)
}

func TestApi_CreateCatalog(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().CreateCatalog().Return(nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "PUT", "/v1/priv/cat1", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog": "cat1"}`, body)
}

func TestApi_CreateCatalog_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().CreateCatalog().Return(errors.New("failed"))

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "PUT", "/v1/priv/cat1", nil)
	assertHTTPInternalError(t, status, body)
}

func TestApi_CreateAnonymousTrack(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().NewTrackID().Return("track100")
	catalog.EXPECT().CreateTrack("track100", gomock.Any(), gomock.Any(), false).Return(true, nil)

	request := priv.CreateTrackRequest{Fingerprint: testFingerprint}
	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "POST", "/v1/priv/cat1", bytes.NewReader(requestBody))
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog": "cat1", "id": "track100"}`, body)
}

func TestApi_CreateTrack(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().CreateTrack("track1", gomock.Any(), gomock.Any(), false).Return(true, nil)

	request := priv.CreateTrackRequest{Fingerprint: testFingerprint}
	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "PUT", "/v1/priv/cat1/track1", bytes.NewReader(requestBody))
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog": "cat1", "id": "track1"}`, body)
}

func TestApi_CreateTrack_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().CreateTrack("track1", gomock.Any(), gomock.Any(), false).Return(false, nil)

	request := priv.CreateTrackRequest{Fingerprint: testFingerprint}
	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "PUT", "/v1/priv/cat1/track1", bytes.NewReader(requestBody))
	assert.Equal(t, http.StatusConflict, status)
	assert.JSONEq(t, `{"status":409,"error":{"type":"duplicate","reason":"Duplicate fingerprint, use allow_duplicate=false if you want to add it anyway"}}`, body)
}

func TestApi_CreateTrack_AllowDuplicate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().CreateTrack("track1", gomock.Any(), gomock.Any(), true).Return(true, nil)

	request := priv.CreateTrackRequest{Fingerprint: testFingerprint, AllowDuplicate: true}
	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "PUT", "/v1/priv/cat1/track1", bytes.NewReader(requestBody))
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog": "cat1", "id": "track1"}`, body)
}

func TestApi_CreateTrack_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().CreateTrack("track1", gomock.Any(), gomock.Any(), false).Return(false, errors.New("failed"))

	request := priv.CreateTrackRequest{Fingerprint: testFingerprint}
	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "PUT", "/v1/priv/cat1/track1", bytes.NewReader(requestBody))
	assertHTTPInternalError(t, status, body)
}

func TestApi_DeleteTrack(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().DeleteTrack("track1").Return(nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "DELETE", "/v1/priv/cat1/track1", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog": "cat1", "id": "track1"}`, body)
}

func TestApi_DeleteTrack_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().DeleteTrack("track1").Return(errors.New("failed"))

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "DELETE", "/v1/priv/cat1/track1", nil)
	assertHTTPInternalError(t, status, body)
}

func TestApi_GetTrack(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().GetTrack("track1").Return(&priv.SearchResults{[]priv.SearchResult{
		{ID: "track1", Metadata: priv.Metadata{"title": "Song title"}},
	}}, nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "GET", "/v1/priv/cat1/track1", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog":"cat1","id":"track1","metadata":{"title":"Song title"}}`, body)
}

func TestApi_GetTrack_NoMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().GetTrack("track1").Return(&priv.SearchResults{[]priv.SearchResult{
		{ID: "track1"},
	}}, nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "GET", "/v1/priv/cat1/track1", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog":"cat1","id":"track1"}`, body)
}

func TestApi_GetCatalog(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().Exists().Return(true, nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "GET", "/v1/priv/cat1", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog":"cat1"}`, body)
}

func TestApi_GetCatalog_DoesNotExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().Exists().Return(false, nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "GET", "/v1/priv/cat1", nil)
	assert.Equal(t, http.StatusNotFound, status)
	assert.JSONEq(t, `{"status":404,"error":{"type":"not_found","reason":"Catalog not found"}}`, body)
}

func TestApi_GetCatalog_ListTracks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().Exists().Return(true, nil)
	catalog.EXPECT().ListTracks("", 100).Return(&priv.ListTracksResult{
		HasMore: true,
		Tracks: []priv.TrackDetails{
			{
				ID:       "track1",
				Metadata: priv.Metadata{"title": "Track 1"},
			},
			{
				ID:       "track2",
				Metadata: priv.Metadata{"title": "Track 2"},
			},
		},
	}, nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "GET", "/v1/priv/cat1?tracks", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog":"cat1","tracks":[{"id":"track1","metadata":{"title":"Track 1"}},{"id":"track2","metadata":{"title":"Track 2"}}],"has_more":true,"cursor":"track2"}`, body)
}

func TestApi_GetCatalog_ListTracks_More(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().Exists().Return(true, nil)
	catalog.EXPECT().ListTracks("track100", 100).Return(&priv.ListTracksResult{
		HasMore: false,
		Tracks: []priv.TrackDetails{
			{
				ID:       "track101",
				Metadata: priv.Metadata{"title": "Track 101"},
			},
		},
	}, nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "GET", "/v1/priv/cat1?tracks&cursor=track100", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog":"cat1","tracks":[{"id":"track101","metadata":{"title":"Track 101"}}],"has_more":false}`, body)
}

func TestApi_GetCatalog_ListTracks_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().Exists().Return(true, nil)
	catalog.EXPECT().ListTracks("", 100).Return(&priv.ListTracksResult{
		HasMore: false,
	}, nil)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "GET", "/v1/priv/cat1?tracks", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog":"cat1","tracks":[],"has_more":false}`, body)
}

func TestApi_Search(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().Search(gomock.Any(), gomock.Any()).DoAndReturn(func(query *chromaprint.Fingerprint, opts *priv.SearchOptions) (*priv.SearchResults, error) {
		results := &priv.SearchResults{
			Results: []priv.SearchResult{
				{
					ID:       "track1",
					Metadata: priv.Metadata{"name": "Track 1"},
					Match: &priv.MatchResult{
						Version:      1,
						Config:       priv.FingerprintConfigs[1],
						MasterLength: 1,
						QueryLength:  1,
						Sections: []priv.MatchingSection{
							{Offset: 0, Start: 0, End: 121},
						},
					},
				},
			},
		}
		return results, nil
	})

	request := priv.SearchRequest{Fingerprint: testFingerprint}
	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "POST", "/v1/priv/cat1/_search", bytes.NewReader(requestBody))
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog": "cat1", "results": [{"id": "track1", "metadata": {"name": "Track 1"}, "match": {"position": 0, "duration": 17.580979}}]}`, body)
}

func TestApi_Search_Stream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, catalog := createMockCatalogService(ctrl)
	catalog.EXPECT().Search(gomock.Any(), gomock.Any()).DoAndReturn(func(query *chromaprint.Fingerprint, opts *priv.SearchOptions) (*priv.SearchResults, error) {
		results := &priv.SearchResults{
			Results: []priv.SearchResult{
				{
					ID:       "track1",
					Metadata: priv.Metadata{"name": "Track 1"},
					Match: &priv.MatchResult{
						Version:      1,
						Config:       priv.FingerprintConfigs[1],
						MasterLength: 1,
						QueryLength:  1,
						Sections: []priv.MatchingSection{
							{Offset: 0, Start: 0, End: 121},
						},
					},
				},
			},
		}
		return results, nil
	})

	request := priv.SearchRequest{Fingerprint: testFingerprint, Stream: true}
	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	api := priv.NewAPI(service)
	status, body := makeRequest(t, api, "POST", "/v1/priv/cat1/_search", bytes.NewReader(requestBody))
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"catalog": "cat1", "results": [{"id": "track1", "metadata": {"name": "Track 1"}, "match": {"position": 0, "duration": 17.580979}}]}`, body)
}

func assertHTTPInternalError(t *testing.T, status int, body string) {
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.JSONEq(t, `{"error": {"type": "internal_error", "reason": "Internal error"}, "status": 500}`, body)
}
