package priv

import (
	"encoding/json"
	"fmt"
	"github.com/acoustid/go-acoustid/chromaprint"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"net/http"
	"sync/atomic"
)

type Error struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type ErrorResponse struct {
	Status int   `json:"status"`
	Error  Error `json:"error"`
}

type API struct {
	service Service
	router  *mux.Router
	Auth    Authenticator
	status  int32
}

func NewAPI(service Service) *API {
	s := &API{service: service}
	s.router = s.createRouter()
	s.Auth = &NoAuth{}
	s.SetHealthStatus(true)
	return s
}

func (s *API) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.router.ServeHTTP(w, req)
}

func (s *API) createRouter() *mux.Router {
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(s.NotFoundHandler)
	router.MethodNotAllowedHandler = http.HandlerFunc(s.MethodNotAllowedHandler)
	router.HandleFunc("/_health", s.HealthHandler)
	router.Handle("/_metrics", promhttp.Handler())
	v1 := router.PathPrefix("/v1/priv").Subrouter()
	v1.Methods(http.MethodGet).Path("").HandlerFunc(s.wrapHandler(s.ListCatalogsHandler))
	v1.Methods(http.MethodGet).Path("/{catalog}").HandlerFunc(s.wrapCatalogHandler(s.GetCatalogHandler))
	v1.Methods(http.MethodPut).Path("/{catalog}").HandlerFunc(s.wrapCatalogHandler(s.CreateCatalogHandler))
	v1.Methods(http.MethodDelete).Path("/{catalog}").HandlerFunc(s.wrapCatalogHandler(s.DeleteCatalogHandler))
	v1.Methods(http.MethodPost).Path("/{catalog}").HandlerFunc(s.wrapCatalogHandler(s.CreateAnonymousTrackHandler))
	v1.Methods(http.MethodPost).Path("/{catalog}/_search").HandlerFunc(s.wrapCatalogHandler(s.SearchHandler))
	v1.Methods(http.MethodGet).Path("/{catalog}/{track}").HandlerFunc(s.wrapTrackHandler(s.GetTrackHandler))
	v1.Methods(http.MethodPut).Path("/{catalog}/{track}").HandlerFunc(s.wrapTrackHandler(s.CreateTrackHandler))
	v1.Methods(http.MethodDelete).Path("/{catalog}/{track}").HandlerFunc(s.wrapTrackHandler(s.DeleteTrackHandler))
	return router
}

func (s *API) NotFoundHandler(w http.ResponseWriter, request *http.Request) {
	writeResponseError(w, http.StatusNotFound, Error{"not_found", "Page not found"})
}

func (s *API) MethodNotAllowedHandler(w http.ResponseWriter, request *http.Request) {
	writeResponseError(w, http.StatusNotFound, Error{"method_not_allowed", "Method not allowed"})
}

func (s *API) wrapHandler(handler func(w http.ResponseWriter, req *http.Request, repo Repository)) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		externalAccountID, err := s.Auth.Authenticate(req)
		if err != nil {
			if errors.Cause(err) == ErrNotAuthorized {
				reason := fmt.Sprintf("Not authorized: %s", err)
				writeResponseError(w, http.StatusUnauthorized, Error{"unauthorized", reason})
				return
			}
			log.Printf("Failed to authenticate account: %v", err)
			writeResponseInternalError(w)
			return
		}
		account, err := s.service.GetAccount(externalAccountID)
		if err != nil {
			log.Printf("Failed to get account: %v", err)
			writeResponseInternalError(w)
			return
		}
		handler(w, req, account.Repository())
	}
}

func (s *API) wrapCatalogHandler(handler func(w http.ResponseWriter, req *http.Request, catalog Catalog)) http.HandlerFunc {
	return s.wrapHandler(func(w http.ResponseWriter, req *http.Request, repo Repository) {
		vars := mux.Vars(req)
		catalogName := vars["catalog"]
		if !IsValidCatalogName(catalogName) {
			writeResponseError(w, http.StatusBadRequest, Error{"invalid_request", "Invalid catalog name"})
			return
		}
		handler(w, req, repo.Catalog(catalogName))
	})
}

func (s *API) wrapTrackHandler(handler func(w http.ResponseWriter, req *http.Request, catalog Catalog, trackID string)) http.HandlerFunc {
	return s.wrapCatalogHandler(func(w http.ResponseWriter, req *http.Request, catalog Catalog) {
		vars := mux.Vars(req)
		trackID := vars["track"]
		if !IsValidTrackID(trackID) {
			writeResponseError(w, http.StatusBadRequest, Error{"invalid_request", "Invalid track ID"})
			return
		}
		handler(w, req, catalog, trackID)
	})
}

func (s *API) SetHealthStatus(status bool) {
	var value int32
	if status {
		value = 1
	}
	atomic.StoreInt32(&s.status, value)
}

func (s *API) HealthHandler(w http.ResponseWriter, req *http.Request) {
	status := atomic.LoadInt32(&s.status)
	if status == 0 {
		writeResponseError(w, http.StatusServiceUnavailable, Error{"unavailable", "Service is unavailable"})
	} else {
		writeResponseOK(w, struct{}{})
	}
}

type ListCatalogsResponse struct {
	Catalogs []ListCatalogsResponseCatalog `json:"catalogs"`
}

type ListCatalogsResponseCatalog struct {
	Catalog string `json:"catalog"`
}

func (s *API) ListCatalogsHandler(w http.ResponseWriter, req *http.Request, repo Repository) {
	catalogs, err := repo.ListCatalogs()
	if err != nil {
		log.Printf("Failed to list catalogs: %v", err)
		writeResponseInternalError(w)
		return
	}

	var resp ListCatalogsResponse
	resp.Catalogs = make([]ListCatalogsResponseCatalog, len(catalogs))
	for i, catalog := range catalogs {
		resp.Catalogs[i].Catalog = catalog.Name()
	}
	writeResponseOK(w, &resp)
}

type CatalogResponse struct {
	Catalog string `json:"catalog"`
}

type ListTracksResponse struct {
	Catalog string                    `json:"catalog"`
	Tracks  []ListTracksResponseTrack `json:"tracks"`
	HasMore bool                      `json:"has_more"`
	Cursor  string                    `json:"cursor,omitempty"`
}

type ListTracksResponseTrack struct {
	ID          string   `json:"id"`
	Metadata    Metadata `json:"metadata,omitempty"`
	Fingerprint string   `json:"fingerprint,omitempty"`
}

func (s *API) GetCatalogHandler(w http.ResponseWriter, request *http.Request, catalog Catalog) {
	exists, err := catalog.Exists()
	if err != nil {
		log.Printf("Failed to get catalog %s: %v", catalog.Name(), err)
		writeResponseInternalError(w)
		return
	}
	if !exists {
		writeResponseError(w, http.StatusNotFound, Error{"not_found", "Catalog not found"})
		return
	}

	query := request.URL.Query()
	if len(query["tracks"]) == 0 {
		writeResponseOK(w, &CatalogResponse{catalog.Name()})
		return
	}

	lastTrackID := query.Get("cursor")
	results, err := catalog.ListTracks(lastTrackID, 100)
	if err != nil {
		log.Printf("Failed to list tracks in catalog %s: %v", catalog.Name(), err)
		writeResponseInternalError(w)
		return
	}

	response := &ListTracksResponse{
		Catalog: catalog.Name(),
		HasMore: results.HasMore,
		Tracks: make([]ListTracksResponseTrack, len(results.Tracks)),
	}

	if results.HasMore {
		response.Cursor = results.Tracks[len(results.Tracks)-1].ID
	}

	for i, track := range results.Tracks {
		response.Tracks[i].ID = track.ID
		response.Tracks[i].Metadata = track.Metadata
	}

	writeResponseOK(w, response)
}

func (s *API) CreateCatalogHandler(w http.ResponseWriter, request *http.Request, catalog Catalog) {
	err := catalog.CreateCatalog()
	if err != nil {
		log.Printf("Failed to create catalog %s: %v", catalog.Name(), err)
		writeResponseInternalError(w)
		return
	}
	writeResponseOK(w, &CatalogResponse{catalog.Name()})
}

func (s *API) DeleteCatalogHandler(w http.ResponseWriter, request *http.Request, catalog Catalog) {
	err := catalog.DeleteCatalog()
	if err != nil {
		log.Printf("Failed to delete catalog %s: %v", catalog.Name(), err)
		writeResponseInternalError(w)
		return
	}
	writeResponseOK(w, &CatalogResponse{catalog.Name()})
}

type TrackResponse struct {
	Catalog  string   `json:"catalog"`
	ID       string   `json:"id"`
	Metadata Metadata `json:"metadata,omitempty"`
}

type CreateTrackRequest struct {
	Fingerprint    string            `json:"fingerprint"`
	Metadata       map[string]string `json:"metadata"`
	AllowDuplicate bool              `json:"allow_duplicate"`
}

func unmarshalRequestJSON(req *http.Request, v interface{}) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return errors.WithMessage(err, "failed to read HTTP body")
	}
	defer req.Body.Close()

	return json.Unmarshal(body, v)
}

func (s *API) CreateAnonymousTrackHandler(w http.ResponseWriter, request *http.Request, catalog Catalog) {
	s.CreateTrackHandler(w, request, catalog, catalog.NewTrackID())
}

func (s *API) CreateTrackHandler(w http.ResponseWriter, request *http.Request, catalog Catalog, trackID string) {
	var data CreateTrackRequest
	err := unmarshalRequestJSON(request, &data)
	if err != nil {
		writeResponseError(w, http.StatusBadRequest, Error{"invalid_request", "Invalid request body"})
		return
	}

	fingerprint, err := chromaprint.ParseFingerprintString(data.Fingerprint)
	if err != nil {
		message := fmt.Sprintf("Invalid request: %v", err)
		writeResponseError(w, http.StatusBadRequest, Error{"invalid_request", message})
		return
	}

	created, err := catalog.CreateTrack(trackID, fingerprint, data.Metadata, data.AllowDuplicate)
	if err != nil {
		log.Printf("Failed to create track %s/%s: %v", catalog.Name(), trackID, err)
		writeResponseInternalError(w)
		return
	}

	if !created {
		message := "Duplicate fingerprint, use allow_duplicate=false if you want to add it anyway"
		writeResponseError(w, http.StatusConflict, Error{"duplicate", message})
		return
	}

	writeResponseOK(w, &TrackResponse{Catalog: catalog.Name(), ID: trackID})
}

func (s *API) DeleteTrackHandler(w http.ResponseWriter, request *http.Request, catalog Catalog, trackID string) {
	err := catalog.DeleteTrack(trackID)
	if err != nil {
		log.Printf("Failed to delete track %s/%s: %v", catalog.Name(), trackID, err)
		writeResponseInternalError(w)
		return
	}

	writeResponseOK(w, &TrackResponse{Catalog: catalog.Name(), ID: trackID})
}

func (s *API) GetTrackHandler(w http.ResponseWriter, request *http.Request, catalog Catalog, trackID string) {
	results, err := catalog.GetTrack(trackID)
	if err != nil {
		log.Printf("Failed to get track %s/%s: %v", catalog.Name(), trackID, err)
		writeResponseInternalError(w)
		return
	}

	if len(results.Results) == 0 {
		message := fmt.Sprintf("Track %s not found", trackID)
		writeResponseError(w, http.StatusNotFound, Error{"not_found", message})
		return
	}

	writeResponseOK(w, &TrackResponse{Catalog: catalog.Name(), ID: trackID, Metadata: results.Results[0].Metadata})
}

type SearchRequest struct {
	Fingerprint string `json:"fingerprint"`
	Stream      bool   `json:"stream"`
}

type SearchResponse struct {
	Catalog string                  `json:"catalog"`
	Results []*SearchResponseResult `json:"results"`
}

type SearchResponseResult struct {
	ID       string                    `json:"id"`
	Match    SearchResponseResultMatch `json:"match"`
	Metadata Metadata                  `json:"metadata,omitempty"`
}

type SearchResponseResultMatch struct {
	Position float64 `json:"position"`
	Duration float64 `json:"duration"`
}

func (s *API) SearchHandler(w http.ResponseWriter, request *http.Request, catalog Catalog) {
	var data SearchRequest
	err := unmarshalRequestJSON(request, &data)
	if err != nil {
		writeResponseError(w, http.StatusBadRequest, Error{"invalid_request", "Invalid request body"})
		return
	}

	fingerprint, err := chromaprint.ParseFingerprintString(data.Fingerprint)
	if err != nil {
		message := fmt.Sprintf("Invalid request: %v", err)
		writeResponseError(w, http.StatusBadRequest, Error{"invalid_request", message})
		return
	}

	if data.Stream && len(fingerprint.Hashes) > 300 {
		writeResponseError(w, http.StatusBadRequest, Error{"invalid_request", "Fingerprint too long for stream search"})
		return
	}

	opts := &SearchOptions{Stream: data.Stream}
	results, err := catalog.Search(fingerprint, opts)
	if err != nil {
		log.Printf("Failed to search in %s: %v", catalog.Name(), err)
		writeResponseInternalError(w)
		return
	}

	response := &SearchResponse{
		Catalog: catalog.Name(),
		Results: make([]*SearchResponseResult, len(results.Results)),
	}
	for i, result := range results.Results {
		response.Results[i] = &SearchResponseResult{
			ID:       result.ID,
			Metadata: result.Metadata,
			Match: SearchResponseResultMatch{
				Position: result.Match.MasterOffset().Seconds(),
				Duration: result.Match.MatchingDuration().Seconds(),
			},
		}
	}
	writeResponseOK(w, response)
}

func writeResponseOK(w http.ResponseWriter, response interface{}) {
	writeResponse(w, http.StatusOK, response)
}

func writeResponseError(w http.ResponseWriter, status int, error Error) {
	response := &ErrorResponse{
		Status: status,
		Error:  error,
	}
	writeResponse(w, status, response)
}

func writeResponseInternalError(w http.ResponseWriter) {
	writeResponseError(w, http.StatusInternalServerError, Error{"internal_error", "Internal error"})
}

func writeResponse(w http.ResponseWriter, status int, response interface{}) {
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v", err)
		writeResponseInternalError(w)
		return
	}
	w.Header().Add("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	w.Write(data)
}
