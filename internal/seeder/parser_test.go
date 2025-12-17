package seeder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/alexivanou/geocity-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCityIDMap(t *testing.T) {
	cities := []model.City{
		{ID: 1, NameDefault: "City1"},
		{ID: 2, NameDefault: "City2"},
		{ID: 3, NameDefault: "City3"},
	}

	ids := CreateCityIDMap(cities)

	assert.True(t, ids[1])
	assert.True(t, ids[2])
	assert.True(t, ids[3])
	assert.False(t, ids[4])
}

func TestCreateCountryCodeMap(t *testing.T) {
	countries := []model.Country{
		{Code: "RU", NameDefault: "Russia"},
		{Code: "US", NameDefault: "United States"},
	}

	codes := CreateCountryCodeMap(countries)

	assert.True(t, codes["RU"])
	assert.True(t, codes["US"])
	assert.False(t, codes["FR"])
}

func TestParser_ParseCountries(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "countryInfo.txt")

	testData := `#ISO	ISO3	ISO-Numeric	fips	Country	Capital	Area(in sq km)	Population	Continent	tld	CurrencyCode	CurrencyName	Phone	Postal Code Format	Postal Code Regex	Languages	geonameid	neighbours	EquivalentFipsCode
AD	AND	020	AN	Andorra	Andorra la Vella	468	84000	EU	.ad	EUR	Euro	376	AD###	^(?:AD)*(\d{3})$	ca	3041565	ES,FR	
AE	ARE	784	AE	United Arab Emirates	Abu Dhabi	82880	4975593	AS	.ae	AED	Dirham	971		^(\\d{4})$	ar-AE,fa,en,hi,ur	290557	SA,OM	`

	err := os.WriteFile(testFile, []byte(testData), 0644)
	assert.NoError(t, err)

	cfg := config.SeederConfig{BatchSize: 100}
	parser := NewParser(tmpDir, cfg)
	countries, err := parser.ParseCountries()

	assert.NoError(t, err)
	assert.Len(t, countries, 2)
	assert.Equal(t, "Andorra", countries[0].NameDefault)
	assert.Equal(t, 3041565, countries[0].GeonameID)
}

func TestParser_ProcessAlternateNames(t *testing.T) {
	// TSV Format:
	// alternateNameId, geonameid, isolanguage, alternate name, isPreferredName, isShortName, isColloquial, isHistoric
	inputData := `
1	100	en	London	0	0	0	0
2	100	ru	Лондон	0	0	0	0
3	100	link	http://wiki...	0	0	0	0
4	100	en	Londres	0	0	1	0
5	100	en	Old London	0	0	0	1
6	100	fr	Londres	0	0	0	0
7	200	en	Paris	1	0	0	0
8	200	en	Parigi	0	0	0	0
`
	// Data explanation:
	// 1: Valid entry (en) for ID 100
	// 2: Valid entry (ru) for ID 100
	// 3: Ignored (link)
	// 4: Ignored (isColloquial = 1)
	// 5: Ignored (isHistoric = 1)
	// 6: Valid (fr), but can be filtered by config
	// 7: Preferred name for ID 200
	// 8: Not preferred for ID 200 (should be ignored/overwritten by entry 7 if map logic works correctly;
	// here 7 comes first so it stays, if 8 was first 7 would overwrite it)

	// Setup: IDs 100 and 200 exist
	cityIDs := map[int]bool{
		100: true,
		200: true,
	}

	// Case 1: All languages allowed
	t.Run("All languages allowed", func(t *testing.T) {
		parser := NewParser("", config.SeederConfig{BatchSize: 10})

		var captured []model.CityTranslation
		callback := func(batch []model.CityTranslation) error {
			captured = append(captured, batch...)
			return nil
		}

		reader := strings.NewReader(inputData)
		err := parser.processAlternateNamesFromReaderWithCountryMapping(
			reader, cityIDs, nil, nil, callback, nil,
		)
		require.NoError(t, err)

		// Expecting:
		// 1 (en), 2 (ru), 6 (fr), 7 (en, pref), 8 (en, not pref) -> 7 and 8 share the key "200:en".
		// File order: 7 (pref), then 8. Code logic: if exists and isPreferred - overwrite.
		// Here 7 comes first, it is preferred. 8 comes second, it is NOT preferred.
		// 7 remains in the map. If 8 were first, 7 would overwrite it.

		assert.Len(t, captured, 4) // 100:en, 100:ru, 100:fr, 200:en

		// Check 200:en -> should be "Paris" (preferred), not "Parigi"
		found200 := false
		for _, c := range captured {
			if c.CityID == 200 && c.Lang == "en" {
				assert.Equal(t, "Paris", c.Name)
				found200 = true
			}
		}
		assert.True(t, found200, "Should contain preferred name for ID 200")
	})

	// Case 2: Language filter
	t.Run("Filtered languages", func(t *testing.T) {
		cfg := config.SeederConfig{
			AllowedLanguages: []string{"en", "ru"},
			BatchSize:        10,
		}
		parser := NewParser("", cfg)

		var captured []model.CityTranslation
		callback := func(batch []model.CityTranslation) error {
			captured = append(captured, batch...)
			return nil
		}

		reader := strings.NewReader(inputData)
		err := parser.processAlternateNamesFromReaderWithCountryMapping(
			reader, cityIDs, nil, nil, callback, nil,
		)
		require.NoError(t, err)

		// Expecting: only en and ru (fr is excluded)
		// 1 (en), 2 (ru), 7 (en)
		assert.Len(t, captured, 3)

		for _, c := range captured {
			assert.NotEqual(t, "fr", c.Lang)
		}
	})
}
