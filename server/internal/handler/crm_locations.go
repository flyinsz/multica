package handler

import (
	"embed"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
)

//go:embed crm_locations_data.json
var crmLocationsFS embed.FS

type crmLocationName struct {
	Code         string            `json:"code"`
	Name         string            `json:"name"`
	Native       string            `json:"native,omitempty"`
	Translations map[string]string `json:"translations,omitempty"`
}

type crmLocationOption struct {
	Code string `json:"code"`
	Name struct {
		En string `json:"en"`
		Zh string `json:"zh"`
	} `json:"name"`
}

type crmLocationStore struct {
	Countries []crmLocationName            `json:"countries"`
	Regions   map[string][]crmLocationName `json:"regions"`
	Cities    map[string][]crmLocationName `json:"cities"`
}

var (
	crmLocationsOnce sync.Once
	crmLocations     crmLocationStore
	crmLocationsErr  error
)

func loadCRMLocations() (crmLocationStore, error) {
	crmLocationsOnce.Do(func() {
		data, err := crmLocationsFS.ReadFile("crm_locations_data.json")
		if err != nil {
			crmLocationsErr = err
			return
		}
		crmLocationsErr = json.Unmarshal(data, &crmLocations)
	})
	return crmLocations, crmLocationsErr
}

func crmLocationOptionFrom(raw crmLocationName) crmLocationOption {
	var opt crmLocationOption
	opt.Code = raw.Code
	opt.Name.En = raw.Name
	opt.Name.Zh = raw.Translations["zh-CN"]
	if opt.Name.Zh == "" {
		opt.Name.Zh = raw.Translations["zh"]
	}
	if opt.Name.Zh == "" {
		opt.Name.Zh = raw.Native
	}
	if opt.Name.Zh == "" {
		opt.Name.Zh = raw.Name
	}
	return opt
}

func crmLocationOptions(raw []crmLocationName) []crmLocationOption {
	items := make([]crmLocationOption, 0, len(raw))
	for _, item := range raw {
		if strings.TrimSpace(item.Code) == "" {
			continue
		}
		items = append(items, crmLocationOptionFrom(item))
	}
	sort.SliceStable(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name.En) < strings.ToLower(items[j].Name.En)
	})
	return items
}

func (h *Handler) ListCRMLocationCountries(w http.ResponseWriter, r *http.Request) {
	store, err := loadCRMLocations()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "crm_locations_unavailable: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"countries": crmLocationOptions(store.Countries)})
}

func (h *Handler) ListCRMLocationRegions(w http.ResponseWriter, r *http.Request) {
	country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
	if country == "" {
		writeJSON(w, http.StatusOK, map[string]any{"regions": []crmLocationOption{}})
		return
	}
	store, err := loadCRMLocations()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "crm_locations_unavailable: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"regions": crmLocationOptions(store.Regions[country])})
}

func (h *Handler) ListCRMLocationCities(w http.ResponseWriter, r *http.Request) {
	country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
	region := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("region")))
	if country == "" || region == "" {
		writeJSON(w, http.StatusOK, map[string]any{"cities": []crmLocationOption{}})
		return
	}
	store, err := loadCRMLocations()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "crm_locations_unavailable: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cities": crmLocationOptions(store.Cities[country+":"+region])})
}
