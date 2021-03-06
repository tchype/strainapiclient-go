// Package strainapiclient is a Go client module for calling
// The Strain API (learn more at https://http://strains.evanbusse.com).
package strainapiclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const baseURLHost string = "strainapi.evanbusse.com"
const baseURL string = "https://" + baseURLHost

// Client represents the interface a Client must implemenet
type Client interface {
	ListAllEffects() ([]Effect, error)
	ListAllFlavors() ([]Flavor, error)
	ListAllStrains() (ListAllStrainsResult, error)
	SearchStrainsByName(name string) (SearchStrainsByNameResults, error)
	SearchStrainsByRace(race Race) (SearchStrainsByRaceResults, error)
	SearchStrainsByFlavor(flavor Flavor) (SearchStrainsByFlavorResults, error)
	SearchStrainsByEffectName(effectName string) (SearchStrainsByEffectNameResults, error)
	GetStrainDescriptionByStrainID(id int) (string, error)
	GetStrainFlavorsByStrainID(id int) ([]Flavor, error)
	GetStrainEffectsByStrainID(id int) (EffectsByEffectType, error)

	// SetHandleResourceRequestFunc sets the function used to handle requests
	// and returns the previous value of the *HandleResourceRequestFunc.
	SetHandleResourceRequestFunc(f HandleResourceRequestFunc) HandleResourceRequestFunc
}

// HandleResourceRequestFunc is the signature of a function that can handle
// a resource request to the client.
type HandleResourceRequestFunc func(resourcePath string) ([]byte, error)

// DefaultClient is the default implementation of a Client for The Strain API
type DefaultClient struct {
	apiKey                     string
	resourceRequestHandlerFunc HandleResourceRequestFunc
}

// NewDefaultClient creates a new DefaultClient with the apiKey passed in.
func NewDefaultClient(apiKey string) *DefaultClient {
	client := &DefaultClient{apiKey: apiKey}
	client.resourceRequestHandlerFunc = simpleHTTPGetForFullPath
	return client
}

// SetHandleResourceRequestFunc sets a new request handler for the DefaultClient
// (including any custom function that matches the HandleResrourceRequestFunc signature)
// and returns the value that was previously used.
func (c *DefaultClient) SetHandleResourceRequestFunc(f HandleResourceRequestFunc) HandleResourceRequestFunc {
	current := c.resourceRequestHandlerFunc
	c.resourceRequestHandlerFunc = f
	return current
}

// simpleHTTPGet is just a simple wrapper for getting basic
// byte slices from an HTTP GET call.
// It uses the base url of the API and appends the string
// passed in to the path (you must add a leading '/').
func (c *DefaultClient) simpleHTTPGet(restOfURLPath string) ([]byte, error) {
	return c.resourceRequestHandlerFunc(baseURL + "/" + c.apiKey + restOfURLPath)
}

// simpleHTTPGetForFullPath is the default implementation of a
// HandleRsourceRequestFunc.  This implementation makes an HTTP(S)
// call to the DefaultClient's API.  You can override this
// implementation by making your own HandleResourceReqeustFunc
// and set it using the SetHandleResourceRequestFunc() function.
func simpleHTTPGetForFullPath(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", path, nil)
	req.Header.Set("Host", baseURLHost)
	req.Header.Set("User-Agent", "strain-api-client-go/v1")

	client := http.Client{
		Timeout: 0,
	}

	resp, err := client.Do(req)
	if err != nil {
		specificError := fmt.Errorf("There was a problem connecting to the api: %s", err)
		return make([]byte, 0), specificError
	}

	defer resp.Body.Close()

	body, bodyErr := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return make([]byte, 0), fmt.Errorf("Status: %d - %s", resp.StatusCode, string(body))
	}

	if bodyErr != nil || err != nil {
		parsingError := fmt.Errorf("There was a problem reading the body of the response: %s", err)
		return make([]byte, 0), parsingError
	}

	return body, nil
}

// CanConnect simply hits the root of the API with your API Key
// and makes sure it gets back the default response from the API.
func (c *DefaultClient) CanConnect() bool {
	// Expected response: Seems legit to me man...
	body, _ := c.simpleHTTPGet("")
	return string(body) == "Seems legit to me man..."
}

// Effect represents the effects that can be experienced when
// consuming a strain.
type Effect struct {
	Name string     `json:"effect"`
	Type EffectType `json:"type"`
}

// EffectType represents the possible types effects can be.
type EffectType string

// The valid vlaues of EffectType
const (
	// EffectTypePositive represents positive effects
	EffectTypePositive EffectType = "positive"
	// EffectTypeNegative represents negative effects
	EffectTypeNegative = "negative"
	// EffectTypeMedical represents possible medical-related effects
	EffectTypeMedical = "medical"
)

// ListAllEffects returns a slice of Effect elements that
// represents all effects that can be experienced.
func (c *DefaultClient) ListAllEffects() ([]Effect, error) {
	effects := make([]Effect, 0)

	allEffectsJSONBytes, err := c.simpleHTTPGet("/searchdata/effects")
	if err != nil {
		return effects, err
	}

	marshallErr := json.Unmarshal(allEffectsJSONBytes, &effects)
	return effects, marshallErr
}

// Flavor represents a componenet of strain flavor.
type Flavor string

// ListAllFlavors returns a slice of Flavor elements that
// represents all flavors of a strain.
func (c *DefaultClient) ListAllFlavors() ([]Flavor, error) {
	flavors := make([]Flavor, 0)

	allFlavorsJSONBytes, err := c.simpleHTTPGet("/searchdata/flavors")
	if err != nil {
		return flavors, err
	}

	marshallErr := json.Unmarshal(allFlavorsJSONBytes, &flavors)
	return flavors, marshallErr
}

// Race indicates the type of strain (Indica, Sativa, Hybrid)
type Race string

const (
	// RaceIndica represents a Race of a strain
	RaceIndica Race = "indica"
	// RaceSativa represents a Race of a strain
	RaceSativa = "sativa"
	// RaceHybrid represents a Race of a strain
	RaceHybrid = "hybrid"
)

// Strain represents a single strain of cannabis and its properites.
type Strain struct {
	Name        string                  `json:"name"`
	ID          int                     `json:"id"`
	Description string                  `json:"desc"`
	Race        Race                    `json:"race"`
	Flavors     []Flavor                `json:"flavors"`
	Effects     map[EffectType][]string `json:"effects"`
}

const strainsBasePath string = "/strains"
const strainSearchBasePath string = strainsBasePath + "/search"

// ListAllStrainsResult represents the results of a strain search
type ListAllStrainsResult map[string]Strain

// ListAllStrains gets a ListAllStrainsResult of all strains
// (please use sparingly, it is expensive to run).
func (c *DefaultClient) ListAllStrains() (ListAllStrainsResult, error) {
	strainsResults := make(ListAllStrainsResult)

	findAllURL := strainSearchBasePath + "/all"
	strainsResultsJSONBytes, err := c.simpleHTTPGet(findAllURL)

	if err != nil {
		return strainsResults, err
	}

	marshallErr := json.Unmarshal(strainsResultsJSONBytes, &strainsResults)

	populateStrainNames(strainsResults)

	return strainsResults, marshallErr
}

// Set the name on each Strain to the name of the key
func populateStrainNames(strains ListAllStrainsResult) {
	for name, strain := range strains {
		strain.Name = name
		// Have to assign it back to the map to make it stick
		strains[name] = strain
	}
}

// SearchStrainsByNameResult represents a single item in the results of a
// SearchStrainsByName call.
type SearchStrainsByNameResult struct {
	Name        string `json:"name"`
	ID          int    `json:"id"`
	Description string `json:"desc"`
	Race        Race   `json:"race"`
}

// SearchStrainsByNameResults is a slice of SearchStrainsByNameResult
// results from a SearchStrainsByName call.
type SearchStrainsByNameResults []SearchStrainsByNameResult

// SearchStrainsByName returns a SearchStrainsByNameResults of all strains matching
// the name passed in.
func (c *DefaultClient) SearchStrainsByName(name string) (SearchStrainsByNameResults, error) {
	strainsResults := make(SearchStrainsByNameResults, 0)

	searchURL := strainSearchBasePath + "/name/" + name
	strainsResultsJSONBytes, err := c.simpleHTTPGet(searchURL)

	if err != nil {
		return strainsResults, err
	}

	marshallErr := json.Unmarshal(strainsResultsJSONBytes, &strainsResults)

	return strainsResults, marshallErr
}

// SearchStrainsByRaceResult represents a single item in the results of a
// SearchStrainsByRace call.
type SearchStrainsByRaceResult struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
	Race Race   `json:"race"`
}

// SearchStrainsByRaceResults is a slice of SearchStrainsByRaceResult
// results from a SearchStrainsByRace call.
type SearchStrainsByRaceResults []SearchStrainsByRaceResult

// SearchStrainsByRace gets a SearchStrainsByRaceResult of all strains matching
// the Race passed in.
func (c *DefaultClient) SearchStrainsByRace(race Race) (SearchStrainsByRaceResults, error) {
	strainsResults := make(SearchStrainsByRaceResults, 0)

	searchURL := strainSearchBasePath + "/race/" + url.PathEscape(string(race))
	strainsResultsJSONBytes, err := c.simpleHTTPGet(searchURL)

	if err != nil {
		return strainsResults, err
	}

	marshallErr := json.Unmarshal(strainsResultsJSONBytes, &strainsResults)

	return strainsResults, marshallErr
}

// SearchStrainsByEffectNameResult represents a single item in the results of a
// SearchStrainsByEffectName call.
type SearchStrainsByEffectNameResult struct {
	Name       string `json:"name"`
	ID         int    `json:"id"`
	Race       Race   `json:"race"`
	EffectName string `json:"effect"`
}

// SearchStrainsByEffectNameResults is a slice of SearchStrainsByEffectResult
// results from a SearchStrainsByEffect call.
type SearchStrainsByEffectNameResults []SearchStrainsByEffectNameResult

// SearchStrainsByEffectName returns a SearchStrainsByEffectNameResults of all strains
// with an effect that matches the Effect passed in.
func (c *DefaultClient) SearchStrainsByEffectName(effectName string) (SearchStrainsByEffectNameResults, error) {
	strainsResults := make(SearchStrainsByEffectNameResults, 0)

	searchURL := strainSearchBasePath + "/effect/" + url.PathEscape(string(effectName))
	strainsResultsJSONBytes, err := c.simpleHTTPGet(searchURL)

	if err != nil {
		return strainsResults, err
	}

	marshallErr := json.Unmarshal(strainsResultsJSONBytes, &strainsResults)

	return strainsResults, marshallErr
}

// SearchStrainsByFlavorResult represents a single item in the results of a
// SearchStrainsByFlavor call.
type SearchStrainsByFlavorResult struct {
	Name   string `json:"name"`
	ID     int    `json:"id"`
	Race   Race   `json:"race"`
	Flavor Flavor `json:"flavor"`
}

// SearchStrainsByFlavorResults is a slice of SearchStrainsByFlavorResult
// results from a SearchStrainsByEffect call.
type SearchStrainsByFlavorResults []SearchStrainsByFlavorResult

// SearchStrainsByFlavor returns a SearchStrainsByFlavorResults of all strains
// with a flavor that matches the Flavor passed in.
func (c *DefaultClient) SearchStrainsByFlavor(flavor Flavor) (SearchStrainsByFlavorResults, error) {
	strainsResults := make(SearchStrainsByFlavorResults, 0)

	searchURL := strainSearchBasePath + "/flavor/" + url.PathEscape(string(flavor))
	strainsResultsJSONBytes, err := c.simpleHTTPGet(searchURL)

	if err != nil {
		return strainsResults, err
	}

	marshallErr := json.Unmarshal(strainsResultsJSONBytes, &strainsResults)

	return strainsResults, marshallErr
}

const strainDataBasePath string = strainsBasePath + "/data"

func (c *DefaultClient) getStrainDataByID(dataElementName string, id int) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%d", strainDataBasePath, dataElementName, id)

	return c.simpleHTTPGet(url)
}

// GetStrainDescriptionByStrainID retrieves the Description field for the
// Strain with the ID passed in.
func (c *DefaultClient) GetStrainDescriptionByStrainID(id int) (string, error) {

	description := ""
	descriptionResultBytes, err := c.getStrainDataByID("desc", id)

	if err != nil {
		return "", fmt.Errorf("Problem getting the description for strain with ID %d: %s", id, err)
	}

	result := make(map[string]string)

	marshallErr := json.Unmarshal(descriptionResultBytes, &result)

	if marshallErr != nil {
		return "", marshallErr
	}

	description = result["desc"]

	if description == "" {
		return "", fmt.Errorf("Unable to find description in result")
	}

	return description, nil
}

// GetStrainFlavorsByStrainID returns a slice of Flavors for
// the Strain of the id passed in.
func (c *DefaultClient) GetStrainFlavorsByStrainID(id int) ([]Flavor, error) {
	flavors := make([]Flavor, 0)

	flavorsResultBytes, err := c.getStrainDataByID("flavors", id)
	if err != nil {
		return flavors, fmt.Errorf("Problem getting flavors for stain with ID %d: %s", id, err)
	}

	marshallErr := json.Unmarshal(flavorsResultBytes, &flavors)
	if marshallErr != nil {
		return flavors, fmt.Errorf("Problem parsing flavors response for string with ID %d: %s\nBytes: %v", id, err, flavorsResultBytes)
	}

	return flavors, nil
}

// EffectsByEffectType represents a map of Effect slices, keyed by EffectType.
type EffectsByEffectType map[EffectType][]Effect

// GetStrainEffectsByStrainID returns an EffectsByEffectType.
// Use EffectTypePositive, EffectTypeNegative, and EffectTypeMedical for the keys
// and the values are a slice of Effect items.
func (c *DefaultClient) GetStrainEffectsByStrainID(id int) (EffectsByEffectType, error) {
	effects := make(EffectsByEffectType)

	effectsResultBytes, err := c.getStrainDataByID("effects", id)
	if err != nil {
		return effects, fmt.Errorf("Problem retrieving effects for Strain with ID %d: %s", id, err)
	}

	marshallErr := json.Unmarshal(effectsResultBytes, &effects)
	if marshallErr != nil {
		return effects, fmt.Errorf("Problem parsing effects for Strain with ID %d: %s", id, marshallErr)
	}

	return effects, nil
}

// MarshalJSON is implemented here becuase the output JSON
// is a JSON obect with the type as the keys and an array
// of effectNames (no EffectTypes since it's the key in the
// map).
func (e EffectsByEffectType) MarshalJSON() ([]byte, error) {
	result := make(map[string][]string)

	for effectType, effects := range e {
		effectNames := make([]string, 0)

		for _, effect := range effects {
			effectNames = append(effectNames, effect.Name)
		}

		result[string(effectType)] = effectNames
	}

	return json.Marshal(result)
}

// UnmarshalJSON is a custom JSON marshaller for this type because
// the source data is missing EffectType on Effect, the Effect is
// just a string in the JSON source, not a object with a name and
// and type.
func (e EffectsByEffectType) UnmarshalJSON(data []byte) error {
	effectsMap := make(map[string][]string)

	marshallErr := json.Unmarshal(data, &effectsMap)
	if marshallErr != nil {
		return fmt.Errorf("Problem parsing effects for Strain: %s", marshallErr)
	}

	for effectTypeString, effectNames := range effectsMap {
		effectType := EffectType(effectTypeString)
		effects := make([]Effect, len(effectNames))

		for index, name := range effectNames {
			effects[index] = Effect{Name: name, Type: effectType}
		}

		e[effectType] = effects
	}

	return nil
}
