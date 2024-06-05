package main

import (
	"math/rand"
	"time"
)

// Define Pokémon struct
type Pokemon struct {
	Name    string
	HP      int
	Attack  int
	Defense int
}

// Function for Pokémon to attack another Pokémon
func (p *Pokemon) AttackEnemy(enemy *Pokemon) {
	damage := p.Attack - enemy.Defense
	if damage < 0 {
		damage = 0
	}
	enemy.HP -= damage
	if enemy.HP < 0 {
		enemy.HP = 0
	}
}

// Function to check if a Pokémon is defeated
func (p *Pokemon) IsDefeated() bool {
	return p.HP <= 0
}

// Generate random Pokémon (for simplicity)
func RandomPokemon() Pokemon {
	rand.Seed(time.Now().UnixNano())
	pokemons := []Pokemon{
		{"Pikachu", 100, 50, 30},
		{"Charmander", 100, 52, 28},
		{"Bulbasaur", 100, 48, 32},
		{"Squirtle", 100, 49, 35},
	}
	return pokemons[rand.Intn(len(pokemons))]
}
