package geo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsFromGironde(t *testing.T) {
	tests := []struct {
		name        string
		city        string
		zipCode     string
		isInGironde bool
	}{
		{
			name:        "From Gironde",
			city:        "Town in Gironde",
			zipCode:     "33333",
			isInGironde: true,
		},
		{
			name:        "Not from Gironde",
			city:        "Town not in Gironde",
			zipCode:     "12345",
			isInGironde: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := Location{City: tc.city, ZipCode: tc.zipCode}
			isInGironde := l.IsInGironde()
			assert.Equal(t, tc.isInGironde, isInGironde)
		})
	}
}
