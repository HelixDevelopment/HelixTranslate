package persistence

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"digital.vasic.translator/internal/verifier"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	m := verifier.Model{
		ID:                   "gpt-4",
		ProviderID:           "openai",
		Name:                 "GPT-4",
		VerificationStatus:   "verified",
		CanSeeCode:           true,
		AffirmativeResponse:  true,
		OverallScore:         0.95,
		ResponsivenessScore:  0.9,
		CodeCapabilityScore:  0.92,
		FeatureRichnessScore: 0.88,
		ReliabilityScore:     0.94,
		Capabilities:         map[string]bool{"streaming": true, "vision": true},
		Pricing:              verifier.PricingInfo{InputTokenCost: 0.03, OutputTokenCost: 0.06, Currency: "USD"},
		LastVerifiedAt:       time.Now().UTC().Truncate(time.Second),
	}

	require.NoError(t, store.SaveModel(m))

	loaded, err := store.GetModel("gpt-4")
	require.NoError(t, err)

	assert.Equal(t, m.ID, loaded.ID)
	assert.Equal(t, m.ProviderID, loaded.ProviderID)
	assert.Equal(t, m.Name, loaded.Name)
	assert.Equal(t, m.VerificationStatus, loaded.VerificationStatus)
	assert.Equal(t, m.CanSeeCode, loaded.CanSeeCode)
	assert.Equal(t, m.AffirmativeResponse, loaded.AffirmativeResponse)
	assert.InDelta(t, m.OverallScore, loaded.OverallScore, 0.001)
	assert.InDelta(t, m.ResponsivenessScore, loaded.ResponsivenessScore, 0.001)
	assert.InDelta(t, m.CodeCapabilityScore, loaded.CodeCapabilityScore, 0.001)
	assert.InDelta(t, m.FeatureRichnessScore, loaded.FeatureRichnessScore, 0.001)
	assert.InDelta(t, m.ReliabilityScore, loaded.ReliabilityScore, 0.001)
	assert.Equal(t, m.Capabilities, loaded.Capabilities)
	assert.Equal(t, m.Pricing, loaded.Pricing)
	assert.Equal(t, m.LastVerifiedAt, loaded.LastVerifiedAt)
}

func TestSQLiteStoreLoadModels(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer store.Close()

	models := []verifier.Model{
		{ID: "m1", ProviderID: "p1", Name: "Model 1", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 0.8},
		{ID: "m2", ProviderID: "p2", Name: "Model 2", VerificationStatus: "verified", CanSeeCode: false, AffirmativeResponse: true, OverallScore: 0.7},
	}
	for _, m := range models {
		require.NoError(t, store.SaveModel(m))
	}

	loaded, err := store.LoadModels()
	require.NoError(t, err)
	require.Len(t, loaded, 2)

	ids := make(map[string]bool)
	for _, m := range loaded {
		ids[m.ID] = true
	}
	assert.True(t, ids["m1"])
	assert.True(t, ids["m2"])
}

func TestSQLiteStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer store.Close()

	m := verifier.Model{ID: "del-me", ProviderID: "p", Name: "Del", VerificationStatus: "verified"}
	require.NoError(t, store.SaveModel(m))

	_, err = store.GetModel("del-me")
	require.NoError(t, err)

	require.NoError(t, store.DeleteModel("del-me"))

	_, err = store.GetModel("del-me")
	require.Error(t, err)

	count, err := store.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSQLiteStoreCount(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer store.Close()

	count, err := store.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	require.NoError(t, store.SaveModel(verifier.Model{ID: "c1", ProviderID: "p", Name: "C1", VerificationStatus: "verified"}))
	count, err = store.Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSQLiteStorePersistenceAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "persist.db")

	// First store instance
	store1, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)
	m := verifier.Model{ID: "persist", ProviderID: "p", Name: "Persist", VerificationStatus: "verified", OverallScore: 0.99}
	require.NoError(t, store1.SaveModel(m))
	require.NoError(t, store1.Close())

	// Re-open
	store2, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store2.Close()

	loaded, err := store2.GetModel("persist")
	require.NoError(t, err)
	assert.Equal(t, "persist", loaded.ID)
	assert.InDelta(t, 0.99, loaded.OverallScore, 0.001)
}

func TestSQLiteStoreUpsert(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer store.Close()

	m1 := verifier.Model{ID: "same", ProviderID: "p", Name: "Old", VerificationStatus: "verified", OverallScore: 0.5}
	require.NoError(t, store.SaveModel(m1))

	m2 := verifier.Model{ID: "same", ProviderID: "p", Name: "New", VerificationStatus: "verified", OverallScore: 0.9}
	require.NoError(t, store.SaveModel(m2))

	loaded, err := store.GetModel("same")
	require.NoError(t, err)
	assert.Equal(t, "New", loaded.Name)
	assert.InDelta(t, 0.9, loaded.OverallScore, 0.001)
}

func TestSQLiteStoreAntiBluffMutation(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer store.Close()

	m := verifier.Model{ID: "mutate", ProviderID: "p", Name: "Mutate", VerificationStatus: "verified", OverallScore: 0.8}
	require.NoError(t, store.SaveModel(m))

	// Mutation: corrupt the database file to simulate disk failure
	require.NoError(t, store.Close())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.db"), []byte("corrupt"), 0644))

	_, err = NewSQLiteStore(filepath.Join(dir, "test.db"))
	require.Error(t, err, "expected error when opening corrupt database")
}
