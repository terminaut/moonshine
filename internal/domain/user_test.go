package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_RegenerateHealth(t *testing.T) {
	tests := []struct {
		name      string
		user      *User
		percent   float64
		expected  int
		expectHp0 bool
	}{
		{
			name:     "normal regeneration",
			user:     &User{Hp: 100, CurrentHp: 50},
			percent:  10,
			expected: 60,
		},
		{
			name:     "full hp returns max",
			user:     &User{Hp: 100, CurrentHp: 100},
			percent:  10,
			expected: 100,
		},
		{
			name:     "over max hp returns max",
			user:     &User{Hp: 100, CurrentHp: 110},
			percent:  10,
			expected: 100,
		},
		{
			name:     "negative hp clamped to 0 then regen",
			user:     &User{Hp: 100, CurrentHp: -10},
			percent:  10,
			expected: 10,
		},
		{
			name:     "min regen floor of 5",
			user:     &User{Hp: 10, CurrentHp: 0},
			percent:  1,
			expected: 5,
		},
		{
			name:     "regen capped at max",
			user:     &User{Hp: 100, CurrentHp: 98},
			percent:  10,
			expected: 100,
		},
		{
			name:     "zero hp user gets min regen",
			user:     &User{Hp: 0, CurrentHp: 0},
			percent:  10,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.RegenerateHealth(tt.percent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUser_ReachedNewLevel(t *testing.T) {
	tests := []struct {
		name     string
		user     *User
		expected bool
	}{
		{
			name: "level 1 with enough exp",
			user: &User{
				Level: 1,
				Exp:   100,
			},
			expected: true,
		},
		{
			name: "level 1 with exact exp",
			user: &User{
				Level: 1,
				Exp:   100,
			},
			expected: true,
		},
		{
			name: "level 1 with not enough exp",
			user: &User{
				Level: 1,
				Exp:   99,
			},
			expected: false,
		},
		{
			name: "level 2 with enough exp",
			user: &User{
				Level: 2,
				Exp:   200,
			},
			expected: true,
		},
		{
			name: "level 10 with enough exp",
			user: &User{
				Level: 10,
				Exp:   20000,
			},
			expected: true,
		},
		{
			name: "level 10 with not enough exp",
			user: &User{
				Level: 10,
				Exp:   19999,
			},
			expected: false,
		},
		{
			name: "level 11 with enough exp",
			user: &User{
				Level: 11,
				Exp:   100000,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.ReachedNewLevel()
			assert.Equal(t, tt.expected, result)
		})
	}
}
