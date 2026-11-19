package geo

import (
	"math"
	"testing"
)

func TestDistanceKmSamePoint(t *testing.T) {
	if got := DistanceKm(34.7, 135.5, 34.7, 135.5); got != 0 {
		t.Fatalf("DistanceKm(same point) = %v, want 0", got)
	}
}

func TestDistanceKmKnownDistance(t *testing.T) {
	got := DistanceKm(0, 0, 0, 1)
	if math.Abs(got-111.195) > 0.2 {
		t.Fatalf("DistanceKm(1 degree longitude) = %v, want about 111.195", got)
	}
}

func TestValidCoordinates(t *testing.T) {
	for _, tc := range []struct {
		lat, lon float64
		want     bool
	}{
		{34.7, 135.5, true},
		{-90, -180, true},
		{90.01, 0, false},
		{0, 180.01, false},
		{math.NaN(), 0, false},
		{0, math.Inf(1), false},
	} {
		if got := ValidCoordinates(tc.lat, tc.lon); got != tc.want {
			t.Errorf("ValidCoordinates(%v, %v) = %v, want %v", tc.lat, tc.lon, got, tc.want)
		}
	}
}
