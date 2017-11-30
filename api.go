package priv

import (
	"encoding/json"
	"fmt"
	"github.com/acoustid/go-acoustid/chromaprint"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
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
}

func NewAPI(service Service) *API {
	s := &API{service: service}
	s.router = s.createRouter()
	return s
}

func (s *API) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.router.ServeHTTP(w, req)
}

func (s *API) createRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/_health", s.HealthHandler)
	router.Methods(http.MethodGet).Path("/{catalog}").HandlerFunc(s.wrapCatalogHandler(s.GetCatalogHandler))
	router.Methods(http.MethodPut).Path("/{catalog}").HandlerFunc(s.wrapCatalogHandler(s.CreateCatalogHandler))
	router.Methods(http.MethodDelete).Path("/{catalog}").HandlerFunc(s.wrapCatalogHandler(s.DeleteCatalogHandler))
	router.Methods(http.MethodPost).Path("/{catalog}").HandlerFunc(s.wrapCatalogHandler(s.CreateAnonymousTrackHandler))
	router.Methods(http.MethodPost).Path("/{catalog}/_search").HandlerFunc(s.wrapCatalogHandler(s.SearchHandler))
	router.Methods(http.MethodPut).Path("/{catalog}/{track}").HandlerFunc(s.wrapTrackHandler(s.CreateTrackHandler))
	router.Methods(http.MethodDelete).Path("/{catalog}/{track}").HandlerFunc(s.wrapTrackHandler(s.DeleteTrackHandler))
	router.NotFoundHandler = http.HandlerFunc(s.NotFoundHandler)
	router.MethodNotAllowedHandler = http.HandlerFunc(s.MethodNotAllowedHandler)
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
		account, err := Authenticate(s.service, req.Header.Get("Authorization"))
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

func (s *API) HealthHandler(w http.ResponseWriter, req *http.Request) {
	writeResponseOK(w, struct{}{})
}

type CatalogResponse struct {
	Catalog string `json:"catalog"`
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
	writeResponseOK(w, &CatalogResponse{catalog.Name()})
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
	Catalog string `json:"catalog"`
	ID      string `json:"id"`
}

type CreateTrackRequest struct {
	Fingerprint string            `json:"fingerprint"`
	Metadata    map[string]string `json:"metadata"`
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

	err = catalog.CreateTrack(trackID, fingerprint, data.Metadata)
	if err != nil {
		log.Printf("Failed to create track %s/%s: %v", catalog.Name(), trackID, err)
		writeResponseInternalError(w)
		return
	}

	writeResponseOK(w, &TrackResponse{catalog.Name(), trackID})
}

func (s *API) DeleteTrackHandler(w http.ResponseWriter, request *http.Request, catalog Catalog, trackID string) {
	err := catalog.DeleteTrack(trackID)
	if err != nil {
		log.Printf("Failed to delete track %s/%s: %v", catalog.Name(), trackID, err)
		writeResponseInternalError(w)
		return
	}

	writeResponseOK(w, &TrackResponse{catalog.Name(), trackID})
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
	ID       string   `json:"id"`
	Metadata Metadata `json:"metadata,omitempty"`
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
		response.Results[i] = &SearchResponseResult{result.ID, result.Metadata}
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
