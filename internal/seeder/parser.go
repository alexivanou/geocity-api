package seeder

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/alexivanou/geocity-api/internal/model"
)

const (
	dataDir = "data"
)

// Parser parses GeoNames data files
type Parser struct {
	dataDir          string
	batchSize        int
	minPopulation    int
	allowedLanguages map[string]bool
}

// NewParser creates a new parser instance with config
func NewParser(dataDir string, seederCfg config.SeederConfig) *Parser {
	allowedLangs := make(map[string]bool)
	for _, lang := range seederCfg.AllowedLanguages {
		allowedLangs[lang] = true
	}

	return &Parser{
		dataDir:          dataDir,
		batchSize:        seederCfg.BatchSize,
		minPopulation:    seederCfg.MinPopulation,
		allowedLanguages: allowedLangs,
	}
}

// ParseCountries parses countryInfo.txt
func (p *Parser) ParseCountries() ([]model.Country, error) {
	filePath := filepath.Join(p.dataDir, "countryInfo.txt")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open countryInfo.txt: %w", err)
	}
	defer file.Close()

	var countries []model.Country
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Parse TSV format
		parts := strings.Split(line, "\t")
		// We need at least column 16 (geonameid)
		if len(parts) < 17 {
			continue
		}

		code := parts[0]
		name := parts[4]
		geonameID, _ := strconv.Atoi(parts[16])

		if code != "" && name != "" {
			countries = append(countries, model.Country{
				Code:        code,
				NameDefault: name,
				GeonameID:   geonameID,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan countryInfo.txt: %w", err)
	}

	return countries, nil
}

// ParseCities parses cities1000.txt and filters by population
func (p *Parser) ParseCities() ([]model.City, error) {
	filePath := filepath.Join(p.dataDir, "cities1000.txt")

	// Check if file is zipped
	zipPath := filepath.Join(p.dataDir, "cities1000.zip")
	if _, err := os.Stat(zipPath); err == nil {
		return p.parseCitiesFromZip(zipPath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cities1000.txt: %w", err)
	}
	defer file.Close()

	return p.parseCitiesFromReader(file)
}

func (p *Parser) parseCitiesFromZip(zipPath string) ([]model.City, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".txt") {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file in zip: %w", err)
			}
			defer rc.Close()
			return p.parseCitiesFromReader(rc)
		}
	}

	return nil, fmt.Errorf("no txt file found in zip")
}

func (p *Parser) parseCitiesFromReader(reader io.Reader) ([]model.City, error) {
	scanner := bufio.NewScanner(reader)
	var cities []model.City

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")

		if len(parts) < 19 {
			continue
		}

		id, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		population, err := strconv.Atoi(parts[14])
		// Use configured minPopulation
		if err != nil || population < p.minPopulation {
			continue
		}

		lat, err := strconv.ParseFloat(parts[4], 64)
		if err != nil {
			continue
		}

		lon, err := strconv.ParseFloat(parts[5], 64)
		if err != nil {
			continue
		}

		var elevation *int
		if parts[15] != "" {
			elev, err := strconv.Atoi(parts[15])
			if err == nil {
				elevation = &elev
			}
		}

		var timezone *string
		if parts[17] != "" {
			timezone = &parts[17]
		}

		city := model.City{
			ID:          id,
			CountryCode: parts[8],
			NameDefault: parts[1],
			Population:  population,
			Lat:         lat,
			Lon:         lon,
			Elevation:   elevation,
			Timezone:    timezone,
		}

		cities = append(cities, city)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan cities: %w", err)
	}

	return cities, nil
}

// ProcessAlternateNames processes alternateNames.txt using streaming approach to avoid OOM
func (p *Parser) ProcessAlternateNames(
	cityIDs map[int]bool,
	countryCodes map[string]bool,
	cityCallback func(batch []model.CityTranslation) error,
) error {
	return p.ProcessAlternateNamesWithCountries(cityIDs, countryCodes, nil, cityCallback, nil)
}

// ProcessAlternateNamesWithCountries processes alternateNames with support for country translations
func (p *Parser) ProcessAlternateNamesWithCountries(
	cityIDs map[int]bool,
	countryCodes map[string]bool,
	geonameIDToCountryCode map[int]string,
	cityCallback func(batch []model.CityTranslation) error,
	countryCallback func(batch []model.CountryTranslation) error,
) error {
	filePath := filepath.Join(p.dataDir, "alternateNames.txt")

	// Check if file is zipped
	zipPath := filepath.Join(p.dataDir, "alternateNames.zip")
	var reader io.Reader
	if _, err := os.Stat(zipPath); err == nil {
		fmt.Printf("DEBUG: Using alternateNames.zip\n")
		r, err := zip.OpenReader(zipPath)
		if err != nil {
			return fmt.Errorf("failed to open zip: %w", err)
		}
		defer r.Close()

		var targetFile *zip.File
		for _, f := range r.File {
			if strings.HasSuffix(f.Name, "alternateNames.txt") || strings.HasSuffix(f.Name, "alternateNamesV2.txt") {
				targetFile = f
				break
			}
			if targetFile == nil && strings.HasSuffix(f.Name, ".txt") {
				targetFile = f
			}
		}

		if targetFile == nil {
			return fmt.Errorf("no alternateNames.txt file found in zip")
		}

		rc, err := targetFile.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}
		defer rc.Close()
		reader = rc
	} else {
		if _, err := os.Stat(filePath); err != nil {
			return fmt.Errorf("alternateNames file not found (checked %s and %s): %w", zipPath, filePath, err)
		}

		fmt.Printf("DEBUG: Using alternateNames.txt\n")
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open alternateNames.txt: %w", err)
		}
		defer file.Close()
		reader = file
	}

	return p.processAlternateNamesFromReaderWithCountryMapping(reader, cityIDs, countryCodes, geonameIDToCountryCode, cityCallback, countryCallback)
}

func (p *Parser) processAlternateNamesFromReaderWithCountryMapping(
	reader io.Reader,
	cityIDs map[int]bool,
	countryCodes map[string]bool,
	geonameIDToCountryCode map[int]string,
	cityCallback func(batch []model.CityTranslation) error,
	countryCallback func(batch []model.CountryTranslation) error,
) error {
	buf := make([]byte, 0, 64*1024)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(buf, 1024*1024)

	batchSize := p.batchSize
	if batchSize <= 0 {
		batchSize = 10000
	}

	cityBatch := make([]model.CityTranslation, 0, batchSize)
	countryBatch := make([]model.CountryTranslation, 0, batchSize)

	// Maps to store index in batch for duplicate handling (prefer preferredName)
	// Key: "id:lang", Value: index in batch
	cityTransMap := make(map[string]int)
	countryTransMap := make(map[string]int)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")

		if len(parts) < 4 {
			continue
		}

		geonameID, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		lang := parts[2]
		name := parts[3]

		if name == "" || lang == "" {
			continue
		}

		// 1. Skip historic and colloquial names
		if len(parts) > 6 && parts[6] == "1" { // isColloquial
			continue
		}
		if len(parts) > 7 && parts[7] == "1" { // isHistoric
			continue
		}

		// 2. Check Preferred Name flag
		isPreferred := false
		if len(parts) > 4 && parts[4] == "1" {
			isPreferred = true
		}

		// 3. Skip technical codes
		if lang == "link" || lang == "post" || lang == "iata" || lang == "icao" || lang == "faac" || lang == "fr_1793" || lang == "abbr" || lang == "wkdt" {
			continue
		}

		if len(lang) > 2 {
			lang = lang[:2]
		}

		// 4. CHECK ALLOWED LANGUAGES
		// If map is empty, allow all. If not empty, check existence.
		if len(p.allowedLanguages) > 0 && !p.allowedLanguages[lang] {
			continue
		}

		// Check if this is a city translation
		if cityIDs[geonameID] {
			key := fmt.Sprintf("%d:%s", geonameID, lang)
			if idx, exists := cityTransMap[key]; exists {
				// If already exists in batch, but this one is preferred, overwrite it
				if isPreferred {
					cityBatch[idx].Name = name
				}
			} else {
				// New entry
				cityBatch = append(cityBatch, model.CityTranslation{
					CityID: geonameID,
					Lang:   lang,
					Name:   name,
				})
				cityTransMap[key] = len(cityBatch) - 1
			}

			if len(cityBatch) >= batchSize {
				if cityCallback != nil {
					if err := cityCallback(cityBatch); err != nil {
						return fmt.Errorf("city callback error: %w", err)
					}
				}
				cityBatch = cityBatch[:0]
				cityTransMap = make(map[string]int)
			}
		}

		// Check if this is a COUNTRY translation
		if countryCallback != nil && geonameIDToCountryCode != nil {
			if countryCode, ok := geonameIDToCountryCode[geonameID]; ok && countryCodes[countryCode] {
				key := fmt.Sprintf("%s:%s", countryCode, lang)
				if idx, exists := countryTransMap[key]; exists {
					// If already exists in batch, but this one is preferred, overwrite it
					if isPreferred {
						countryBatch[idx].Name = name
					}
				} else {
					countryBatch = append(countryBatch, model.CountryTranslation{
						CountryCode: countryCode,
						Lang:        lang,
						Name:        name,
					})
					countryTransMap[key] = len(countryBatch) - 1
				}

				if len(countryBatch) >= batchSize {
					if err := countryCallback(countryBatch); err != nil {
						return fmt.Errorf("country callback error: %w", err)
					}
					countryBatch = countryBatch[:0]
					countryTransMap = make(map[string]int)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan alternateNames: %w", err)
	}

	if len(cityBatch) > 0 && cityCallback != nil {
		if err := cityCallback(cityBatch); err != nil {
			return fmt.Errorf("city callback error: %w", err)
		}
	}

	if len(countryBatch) > 0 && countryCallback != nil {
		if err := countryCallback(countryBatch); err != nil {
			return fmt.Errorf("country callback error: %w", err)
		}
	}

	return nil
}

// CreateCountryGeonameIDMap creates a mapping from Country GeonameID to Country Code
func CreateCountryGeonameIDMap(countries []model.Country) map[int]string {
	m := make(map[int]string)
	for _, country := range countries {
		if country.GeonameID != 0 {
			m[country.GeonameID] = country.Code
		}
	}
	return m
}

// CreateCityIDMap creates a map of city IDs from cities slice
func CreateCityIDMap(cities []model.City) map[int]bool {
	m := make(map[int]bool)
	for _, city := range cities {
		m[city.ID] = true
	}
	return m
}

// CreateCountryCodeMap creates a map of country codes
func CreateCountryCodeMap(countries []model.Country) map[string]bool {
	m := make(map[string]bool)
	for _, country := range countries {
		m[country.Code] = true
	}
	return m
}
