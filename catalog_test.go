package priv

import (
	"fmt"
	"github.com/acoustid/go-acoustid/chromaprint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"testing"
	"time"
)

const TestFingerprint = "AQADtFKYSFKYofGJj0IOUTRy_AgTch1axYidILR0mFENmcdxfEiL9jiuH8089EJ7-B3yQexzVFWOboeI60h_HHWMHiZ3hCwLXTzy4JTx" +
	"RsfX4cqI45IpInTCIL1x9EZEbcd7tJVhDfrxwzt8HD3-D9p2XDq0D0cY0agV_EKL78dPPBeC7byQv0IdHUdzdD_wO8g5QeOPtBX66EFn" +
	"2Jpx5Ucz_Th2ovkMPrgaycgOGVtjI19x_DiR_mgrXDmuwKWsIv_x7JiYUQR7_Iavow_2odKP_fiO-MEvQlx19FnBG_mOMll05AqNLc-h" +
	"bQ--o8yRTsd1pA-mZ0e3mWiiQC-LH_6OH2iiD1Ke4gi_ox9uobkQ6Tl0I0-NnsThLitu3MOO0FPQ6THUH_5E_AhtnFJS_GhCH31H_Dic" +
	"B5dz_AgfQ7RY5PDhowZRnRNcGzkh68gP_8F3IdmPIs9BHTcM5seRIMev4z7OFttxhfhwajeOXMfVEOUG8Tz27XiO7MY7ZD8uQh9y_fjx" +
	"3wjTY1egb2gelag0HjoR8oftBhcefIyOMr2IioePkCGhWA8aH5F-VESjbcfMB4-E5iFy6Ece_LBwEYeP_Ejg40d6oZ4OnUf4ozlefBR6" +
	"-LhcYceP_0jOIj3OC02DS_OCvgijcIGuHHmGno6GaNIhpLqM6_Bhq0Sv7DjyF8x2lMH15vghviK-4z-yD3fQPKh9vNuxH3lnNKeTorYc" +
	"6Dj2LJD0RDiHQx1n_AtCNM9y_AeOMHl0aPvxB_mCZkH_4D_OKbgUHF44MXiXbjgSfkFMSTib0DC3J0F_NA-qEs2yEBe7FT_CHDV--IYU" +
	"Hs2XYsetRIEmMyOcNNHh87gO8cuhAc2F6svx40e-E11OpPnRZ4eu4zmaF2m05OiSG-cnNDm6J8dJHX06JMuk7ESow8tTfApeb4arzEh_" +
	"-OGhOWFwaceTB-qPMDvKRHTwJKvwP0gfRSJ0Pcjl4zqLo_5wCn7QBx8-6UPyHDnx8OgZG01tHHuMq3iFpEqUCzmdBOXxHFeLZ8bNw98Q" +
	"rguSK8EvXMsjHMcV9D2aOsh3JI9yXAqOXgZ15uiTB-9x9EHzzDg5lGSKRgmP5EmOPjqyNUGPisoVNL-wH_mhL9iRS3iT5egJP8JzIe8F" +
	"fTpyxnjwH9Y4fCniQw9cYRfSUReuBL_wJBieHXlwcdBz5BROpEdJ5oGmLXuQTsePHU8UeMSPNP4ILWqeI24SC9dw35gzouCTIjoH8Qgr" +
	"4SeeH_6HG3kenImPIzeaS8kxfT544tFpTMkQkoeeD6FFw9fxJNgdlNmPJzdOpVKCLLkOKVdw4ikiZcm4ojyO_fgSBpcSC25WxM4gckf-" +
	"hMI5dnBTEsfT4XmKZBk5RKtQKTuchDR-45wGGx8e6Sh9pDeS44tywb6I77he4T969MUd2EL1I0dXGuIZBg_uw-KJssKPn8LRw5wE7T2-" +
	"w1dwZg9-Dc22QVtzIZ-OJnTQZ0c7Hc2N96h4w01yIdwLLTnCZ8dP4a3x6KgoxmiO5MgT6jixEzfGo4puXIK7E3kC7dmRn8ITCZ-M47hx" +
	"o7glhJFe6IyOM0ZMBd4iNAzq6_gHN-GC-If-EOl_XDnew5MefMe_IxnXIMwSURQePDInPJ0wXsTJ5GAfI3mOXBb6LGqGBieyyiH8JDyU" +
	"R5SOJ9IDLXF29FmgZ8eP3BlRJ9nh66ge3Me4Pcgt4XGh5VHR4ypHzBN3tBMJM3CeaMjLoK2OTyGehcfnSzg7NBmfoCyLyC90kni6BnmS" +
	"D5UL64djnojELMsU6MrxH-JDBHs44zrxHld8cNaFHw0iyYd6pbhVhO_gZzkuZQ9yHqrI4p2JvvgT9IzQZHE4PMeTRzjCfDlKicVTB4dP" +
	"47kQboeIysJLhJRcNFEWJceL55h9tCOohskxWcmNf_hxhixOHI1nodZ-zN3xhKzBH7-KJA_C68GZZPLwo3pwTEq3wwuX4xL6vvgjJM8R" +
	"8fhzuE2Oo6SOqznK45SRa0imH80ZNbg-NA9qdngTTIt6wlG2IYf2Is6kq3iPH3lGD9S6fPih78WPk1Cz9Zjj4mMTodYj4WQOceHQB6EV" +
	"TmjSHE-U40fnFc2X49GOax92Fem04ZuW4ccf9EcTMcePHiJXY_pRH6k9fPAX9CeaH6U8vEI-EWIcCTk54sdlEf3hMVKFHz_yELIRb0N_" +
	"VCLxSIk8IUwmLUFy6UKfjfDRh8bxLHmQXDsikxSaEwx-7NGDXBaSKYdzKtidg_rxXJgSdXnwnMOPpg-6Hnd0o4dRJ6SgJwqZYMqO_uBF" +
	"jLFS9GcI55eQHwm3HFOzCCU_MDl-dLeChJEcIzweJahsEmF0BcmVg9Gi7IMjJfvx5Hh6DX5-PBGzKPCUUmAu9HkGnxnSE4qcJRSaLkd1" +
	"PEfiF1_wDO8VNHvQMsaJ6EfjBLpO-DoeHXqkGJa6Q7mR-yrMJId7Iu-DQzWJUI8y4j2eJTqa5ZjYK_jB2IGPI40OUU2OLDpO9Md2phKq" +
	"9sA39CLx5-g9XGi-o8bESC-a8UpxXQEfEopOND_-H29ynFaCSj-8RD2muDhy5UguHdSzjPh-AADIcUZQkaBQADDglAFAMAcJQgoQgpAQ" +
	"EDiCkXLISIQAJKARAIgxVjIjAFDCKIEEMUgbRAgRjgBAAAAaCQSccVdoJAhwBggmIJDEIUKAI8IBAwQBghFoAFMOAAgMAYAEQZwVChJg" +
	"BRIIImMsIgIJAaxATAAjxREEAIwQIkwwJQgiEBEgBLGkPAKEIQBJAhiQSAABBGAGKEWAAIQCIxiByAghwCRAMMEcoQIpBQkghlmNDcDC" +
	"WGAIQc5SRY0QBhCitCFAEGQEM0RZIKQzGBCAhIICEMIEgoQAIQBShgjGABAMGCYQEI4IgqgAAgFChQBKIMKQ0kAJQZoARAJBCCCCOkWE" +
	"QEAQJCABCAAjgQACCBEEQAo4QAUDxCoEBEBCKAOIBIgwIRARhAiGnCCKMIkYUIQwIQhrCDBgASPgAECIEEgBBowzSABgHFFKCAWIAIQJ" +
	"Y4RiwABHkCaGOMWMBIgIBwBQjkKiCCECEWOECAoQAABGighEgCFAAqIQAYYRgIADBBKkBEzKIQEgcKAAxZBDkDiAAFDGEgAAFAIBwIBT" +
	"wDBjhJMAGEUIYIIQIoBRRAFCkFDACAQoUkRKAhgDEgBDmLVIISCAYUIQBgAQAgiEgENIMiDIA4IJAZgUFCAigFECKEKNI04IYRRTFhnC" +
	"hBBECCIEIgoQIgBCiCDDBCJECEGEQAAAQYwgBiHACBNCAAQCE8ox6ogAiBFBCEOAIGGMEEYYgJSEwhEgmDEKDAEQUgZBxCwAUFFAyBBA" +
	"IScAKMQoBBQxBEhEGBMCGAOMAUQIKZiQQhHEECGAIACEUUQQQJFUDAhgBGcAECA5NIYoZ4QDlACgFBTAAWCAFQIJIYRhxjFgCTCQAQAA" +
	"M4ABIogWCCgiGSAOIQCJAMAAZBARDggGhAEBGWAYoA5AYCQhBjBhgDGICUcIMkQRBwgggjACEDMMQcAAAA"

const TestFingerprintQuery = "AQAAO4wiRdoiBe5y9Ds6H45WHV5y9NiF3qCO65GRfTiUdTTyqMBP5PAftBWuHEfT5SpCq8d3zDfaueB0wxeeox9GHdMffMcf5Bf06kHz" +
	"rOCN_CiTRUf-YrxiQtR89DtRHumS4zrSY7p0dFsKRzk0vviP5uhvHE0-MHwKQgQBQENFAEDCWIERIEAK4YRwRDpAlBAOAAIAEAAxIggB" +
	"wAgAASLAEYeAUMQZAQ"

const TestFingerprintQuery2 = "AQAAOxyVjVLwobmIPkfOQZeQH90OPzCPWch_iMYr40KpK7guXDN2aHmNetBXXC_Qr2haGXrwI8zR6cd3vKgP-zSOHJqPH9fxHD0aMjr-" +
	"4whvHOcRNovwD5p7NDdyWTiLH89wPUWIhocOhAgRRAqFGQAEQWUQMcAQp4iTCghHjBPCAAoMcgYRBABo4AGFBAIOQUEAUQA"

func TestCatalog_Name(t *testing.T) {
	repo := getTestRepository(t, connectToDB(t))
	catalog := repo.Catalog("cat1")
	assert.Equal(t, "cat1", catalog.Name())
}

func TestCatalog_CreateCatalog(t *testing.T) {
	catalog := getTestCatalog(t, false)

	err := catalog.CreateCatalog()
	assert.NoError(t, err)
}

func TestCatalog_CreateCatalog_AlreadyExists(t *testing.T) {
	catalog := getTestCatalog(t, true)

	err := catalog.CreateCatalog()
	assert.NoError(t, err)
}

func TestCatalog_DeleteCatalog(t *testing.T) {
	catalog := getTestCatalog(t, true)

	err := catalog.DeleteCatalog()
	assert.NoError(t, err)
}

func TestCatalog_DeleteCatalog_DoesNotExist(t *testing.T) {
	catalog := getTestCatalog(t, false)

	err := catalog.DeleteCatalog()
	assert.NoError(t, err)
}

func getTestCatalog(t *testing.T, create bool) Catalog {
	repo := getTestRepository(t, connectToDB(t))
	name := fmt.Sprintf("cat_%d", rand.Uint32())
	catalog := repo.Catalog(name)
	if create {
		err := catalog.CreateCatalog()
		require.NoError(t, err)
	}
	return catalog
}

func TestCatalog_CreateTrack(t *testing.T) {
	catalog := getTestCatalog(t, true)

	fp, err := chromaprint.ParseFingerprintString(TestFingerprint)
	require.NoError(t, err)
	changed, err := catalog.CreateTrack("fp1", fp, Metadata{"name": "Track 1"}, false)
	assert.NoError(t, err)
	assert.Equal(t, true, changed)
}

func TestCatalog_CreateTrack_DisallowDuplicate(t *testing.T) {
	catalog := getTestCatalog(t, true)

	fp, err := chromaprint.ParseFingerprintString(TestFingerprint)
	require.NoError(t, err)
	changed, err := catalog.CreateTrack("fp1", fp, Metadata{"name": "Track 1"}, false)
	assert.NoError(t, err)
	assert.Equal(t, true, changed)

	changed, err = catalog.CreateTrack("fp2", fp, Metadata{"name": "Track 2"}, false)
	assert.NoError(t, err)
	assert.Equal(t, false, changed)
}

func TestCatalog_CreateTrack_AllowDuplicate(t *testing.T) {
	catalog := getTestCatalog(t, true)

	fp, err := chromaprint.ParseFingerprintString(TestFingerprint)
	require.NoError(t, err)
	changed, err := catalog.CreateTrack("fp1", fp, Metadata{"name": "Track 1"}, true)
	assert.NoError(t, err)
	assert.Equal(t, true, changed)

	changed, err = catalog.CreateTrack("fp2", fp, Metadata{"name": "Track 2"}, true)
	assert.NoError(t, err)
	assert.Equal(t, true, changed)
}

func TestCatalog_CreateTrack_Update(t *testing.T) {
	catalog := getTestCatalog(t, true)

	fp, err := chromaprint.ParseFingerprintString(TestFingerprint)
	require.NoError(t, err)
	changed, err := catalog.CreateTrack("fp1", fp, Metadata{"name": "Track 1"}, false)
	assert.NoError(t, err)
	assert.Equal(t, true, changed)

	fp2, err := chromaprint.ParseFingerprintString(TestFingerprintQuery)
	require.NoError(t, err)
	changed, err = catalog.CreateTrack("fp1", fp2, Metadata{"name": "Track 1.2"}, false)
	assert.NoError(t, err)
	assert.Equal(t, true, changed)
}

func TestCatalog_CreateTrack_CatalogDoesNotExist(t *testing.T) {
	catalog := getTestCatalog(t, false)

	fp, err := chromaprint.ParseFingerprintString(TestFingerprint)
	require.NoError(t, err)
	changed, err := catalog.CreateTrack("fp1", fp, nil, false)
	assert.NoError(t, err)
	assert.Equal(t, true, changed)
}

func TestCatalog_DeleteTrack(t *testing.T) {
	catalog := getTestCatalog(t, true)

	fp, err := chromaprint.ParseFingerprintString(TestFingerprint)
	require.NoError(t, err)
	_, err = catalog.CreateTrack("fp1", fp, nil, false)
	require.NoError(t, err)

	err = catalog.DeleteTrack("fp1")
	assert.NoError(t, err)
}

func TestCatalog_DeleteTrack_DoesNotExist(t *testing.T) {
	catalog := getTestCatalog(t, true)

	err := catalog.DeleteTrack("fp1")
	assert.NoError(t, err)
}

func TestCatalog_DeleteTrack_CatalogDoesNotExist(t *testing.T) {
	catalog := getTestCatalog(t, false)

	err := catalog.DeleteTrack("fp1")
	assert.NoError(t, err)
}

func TestCatalog_GetTrack(t *testing.T) {
	catalog := getTestCatalog(t, true)

	metadata := Metadata{"name": "Track 1"}

	fp, err := chromaprint.ParseFingerprintString(TestFingerprint)
	require.NoError(t, err)
	_, err = catalog.CreateTrack("fp1", fp, metadata, false)
	require.NoError(t, err)

	results, err := catalog.GetTrack("fp1")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(results.Results))
	assert.Equal(t, "fp1", results.Results[0].ID)
	assert.Equal(t, metadata, results.Results[0].Metadata)
}

func TestCatalog_GetTrack_DoesNotExist(t *testing.T) {
	catalog := getTestCatalog(t, true)

	results, err := catalog.GetTrack("fp1")
	assert.NoError(t, err)
	assert.Empty(t, results.Results)
}

func TestCatalog_Search_NoStream_NoMatch1(t *testing.T) {
	catalog := getTestCatalog(t, true)

	masterID := "t1"
	masterFP := loadTestFingerprint(t, "calibre_sunrise")
	masterMetadata := Metadata{"title": "Sunrise", "artist": "Calibre"}
	_, err := catalog.CreateTrack(masterID, masterFP, masterMetadata, false)
	require.NoError(t, err)

	queryFP := loadTestFingerprint(t, "radio1_1_ad")
	results, err := catalog.Search(queryFP, &SearchOptions{Stream: false})
	if assert.NoError(t, err) {
		if assert.NotNil(t, results) {
			assert.Empty(t, results.Results)
		}
	}
}

func TestCatalog_Search_Stream_NoMatch(t *testing.T) {
	catalog := getTestCatalog(t, true)

	masterID := "t1"
	masterFP := loadTestFingerprint(t, "calibre_sunrise")
	masterMetadata := Metadata{"title": "Sunrise", "artist": "Calibre"}
	_, err := catalog.CreateTrack(masterID, masterFP, masterMetadata, false)
	require.NoError(t, err)

	queryFP := loadTestFingerprint(t, "radio1_1_ad")
	results, err := catalog.Search(queryFP, &SearchOptions{Stream: true})
	if assert.NoError(t, err) {
		if assert.NotNil(t, results) {
			assert.Empty(t, results.Results)
		}
	}
}

func TestCatalog_Search_Stream_PartialMatch(t *testing.T) {
	catalog := getTestCatalog(t, true)

	masterID := "t1"
	masterFP := loadTestFingerprint(t, "calibre_sunrise")
	masterMetadata := Metadata{"title": "Sunrise", "artist": "Calibre"}
	_, err := catalog.CreateTrack(masterID, masterFP, masterMetadata, false)
	require.NoError(t, err)

	queryFP := loadTestFingerprint(t, "radio1_2_ad_and_calibre_sunshine")
	results, err := catalog.Search(queryFP, &SearchOptions{Stream: true})
	if assert.NoError(t, err) {
		if assert.NotNil(t, results) {
			if assert.NotEmpty(t, results.Results) {
				assert.Equal(t, 1, len(results.Results))
				assert.Equal(t, masterID, results.Results[0].ID)
				assert.Equal(t, masterMetadata, results.Results[0].Metadata)
				assert.Equal(t, "12.876237s", results.Results[0].Match.MatchingDuration().String())
			}
		}
	}
}

func TestCatalog_Search_Stream_FullMatch(t *testing.T) {
	catalog := getTestCatalog(t, true)

	masterID := "t1"
	masterFP := loadTestFingerprint(t, "calibre_sunrise")
	masterMetadata := Metadata{"title": "Sunrise", "artist": "Calibre"}
	_, err := catalog.CreateTrack(masterID, masterFP, masterMetadata, false)
	require.NoError(t, err)

	queryFP := loadTestFingerprint(t, "radio1_3_calibre_sunshine")
	results, err := catalog.Search(queryFP, &SearchOptions{Stream: true})
	if assert.NoError(t, err) {
		if assert.NotNil(t, results) {
			if assert.NotEmpty(t, results.Results) {
				assert.Equal(t, 1, len(results.Results))
				assert.Equal(t, masterID, results.Results[0].ID)
				assert.Equal(t, masterMetadata, results.Results[0].Metadata)
				assert.Equal(t, "17.580979s", results.Results[0].Match.MatchingDuration().String())
			}
		}
	}
}

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UTC().UnixNano())
	os.Exit(m.Run())
}
