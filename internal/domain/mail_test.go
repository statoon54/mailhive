package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlexTime_RFC3339(t *testing.T) {
	input := `"2025-01-15T10:30:00Z"`
	var ft FlexTime
	require.NoError(t, json.Unmarshal([]byte(input), &ft))
	assert.Equal(t, 2025, time.Time(ft).Year())
	assert.Equal(t, time.January, time.Time(ft).Month())
	assert.Equal(t, 15, time.Time(ft).Day())
}

func TestFlexTime_DateTimeT(t *testing.T) {
	input := `"2025-06-01T14:00:00"`
	var ft FlexTime
	require.NoError(t, json.Unmarshal([]byte(input), &ft))
	assert.Equal(t, 14, time.Time(ft).Hour())
}

func TestFlexTime_DateTimeSpace(t *testing.T) {
	input := `"2025-03-20 09:30:00"`
	var ft FlexTime
	require.NoError(t, json.Unmarshal([]byte(input), &ft))
	assert.Equal(t, 9, time.Time(ft).Hour())
	assert.Equal(t, 30, time.Time(ft).Minute())
}

func TestFlexTime_DateOnly(t *testing.T) {
	input := `"2025-12-25"`
	var ft FlexTime
	require.NoError(t, json.Unmarshal([]byte(input), &ft))
	assert.Equal(t, 25, time.Time(ft).Day())
	assert.Equal(t, 0, time.Time(ft).Hour())
}

func TestFlexTime_WithTimezone(t *testing.T) {
	input := `"2025-01-15T10:30:00Z"`
	var ft FlexTime
	require.NoError(t, json.Unmarshal([]byte(input), &ft))
	assert.True(t, time.Time(ft).Location() == time.UTC)
}

func TestFlexTime_AppTimezone(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	require.NoError(t, err)
	old := AppTimezone
	AppTimezone = paris
	defer func() { AppTimezone = old }()

	input := `"2025-06-15T14:00:00"`
	var ft FlexTime
	require.NoError(t, json.Unmarshal([]byte(input), &ft))
	assert.Equal(t, paris, time.Time(ft).Location())
}

func TestFlexTime_Invalid(t *testing.T) {
	input := `"not-a-date"`
	var ft FlexTime
	err := json.Unmarshal([]byte(input), &ft)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "format de date non reconnu")
}

func TestFlexTime_Null(t *testing.T) {
	input := `"null"`
	var ft FlexTime
	require.NoError(t, json.Unmarshal([]byte(input), &ft))
	assert.True(t, time.Time(ft).IsZero())
}

func TestFlexTime_Empty(t *testing.T) {
	input := `""`
	var ft FlexTime
	require.NoError(t, json.Unmarshal([]byte(input), &ft))
	assert.True(t, time.Time(ft).IsZero())
}

func TestFlexTime_TimePtr_Zero(t *testing.T) {
	ft := FlexTime(time.Time{})
	assert.Nil(t, ft.TimePtr())
}

func TestFlexTime_TimePtr_NonZero(t *testing.T) {
	now := time.Now()
	ft := FlexTime(now)
	ptr := ft.TimePtr()
	require.NotNil(t, ptr)
	assert.Equal(t, now.Unix(), ptr.Unix())
}

func TestFlexTime_MarshalJSON_RoundTrip(t *testing.T) {
	original := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	ft := FlexTime(original)

	b, err := json.Marshal(ft)
	require.NoError(t, err)

	var ft2 FlexTime
	require.NoError(t, json.Unmarshal(b, &ft2))
	assert.Equal(t, original.Unix(), time.Time(ft2).Unix())
}
