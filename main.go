package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/eymardfreire/pokedexcli/internal/pokecache"
)

type cliCommand struct {
	name        string
	description string
	callback    func(cfg *config, args []string) error
}

type config struct {
	Next     string
	Previous string
	Current  []string
	Cache    *pokecache.Cache
	Caught   map[string]Pokemon
}

type Pokemon struct {
	Name           string `json:"name"`
	BaseExperience int    `json:"base_experience"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
	Stats          []Stat `json:"stats"`
	Types          []Type `json:"types"`
}

type Stat struct {
	BaseStat int `json:"base_stat"`
	Stat     struct {
		Name string `json:"name"`
	} `json:"stat"`
}

type Type struct {
	Type struct {
		Name string `json:"name"`
	} `json:"type"`
}

func commandHelp(cfg *config, args []string) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println("help: Displays a help message")
	fmt.Println("exit: Exit the Pokedex")
	fmt.Println("map: Display the next 20 location areas")
	fmt.Println("mapb: Display the previous 20 location areas")
	fmt.Println("explore <area_name>: Explore a specific location area")
	fmt.Println("catch <pokemon_name>: Try to catch a Pokémon")
	fmt.Println("inspect <pokemon_name>: Inspect a caught Pokémon")
	fmt.Println("pokedex: List all caught Pokémon")
	return nil
}

func commandExit(cfg *config, args []string) error {
	fmt.Println("Exiting Pokedex...")
	os.Exit(0)
	return nil
}

func commandMap(cfg *config, args []string) error {
	if cfg.Next == "" {
		cfg.Next = "https://pokeapi.co/api/v2/location-area/"
	}
	return fetchLocations(cfg, cfg.Next)
}

func commandMapB(cfg *config, args []string) error {
	if cfg.Previous == "" {
		fmt.Println("No previous locations to display.")
		return nil
	}
	return fetchLocations(cfg, cfg.Previous)
}

func commandExplore(cfg *config, args []string) error {
	if len(args) < 1 {
		fmt.Println("Please specify a location area to explore.")
		return nil
	}
	areaName := args[0]
	url := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s/", areaName)
	return fetchLocationDetails(cfg, url)
}

func commandCatch(cfg *config, args []string) error {
	if len(args) < 1 {
		fmt.Println("Please specify a Pokémon to catch.")
		return nil
	}
	pokemonName := args[0]
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s/", pokemonName)
	return catchPokemon(cfg, url)
}

func commandInspect(cfg *config, args []string) error {
	if len(args) < 1 {
		fmt.Println("Please specify a Pokémon to inspect.")
		return nil
	}
	pokemonName := args[0]
	if pokemon, exists := cfg.Caught[pokemonName]; exists {
		printPokemonDetails(pokemon)
	} else {
		fmt.Println("You have not caught that Pokémon.")
	}
	return nil
}

func commandPokedex(cfg *config, args []string) error {
	fmt.Println("Your Pokedex:")
	for name := range cfg.Caught {
		fmt.Printf(" - %s\n", name)
	}
	return nil
}

func fetchLocations(cfg *config, url string) error {
	if data, ok := cfg.Cache.Get(url); ok {
		fmt.Println("Using cached data")
		return displayLocations(data, cfg)
	}

	fmt.Println("Fetching new data")
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	cfg.Cache.Add(url, body)
	return displayLocations(body, cfg)
}

func fetchLocationDetails(cfg *config, url string) error {
	if data, ok := cfg.Cache.Get(url); ok {
		fmt.Println("Using cached data")
		return displayPokemon(data)
	}

	fmt.Println("Fetching new data")
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	cfg.Cache.Add(url, body)
	return displayPokemon(body)
}

func catchPokemon(cfg *config, url string) error {
	if data, ok := cfg.Cache.Get(url); ok {
		return attemptCatch(cfg, data)
	}

	fmt.Println("Fetching new data")
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	cfg.Cache.Add(url, body)
	return attemptCatch(cfg, body)
}

func attemptCatch(cfg *config, data []byte) error {
	var pokemon Pokemon
	err := json.Unmarshal(data, &pokemon)
	if err != nil {
		return err
	}

	fmt.Printf("Throwing a Pokeball at %s...\n", pokemon.Name)
	rand.Seed(time.Now().UnixNano())
	chance := rand.Intn(100)
	if chance < 50 { // This can be adjusted based on base experience or other logic
		fmt.Printf("%s escaped!\n", pokemon.Name)
		return nil
	}

	fmt.Printf("%s was caught!\n", pokemon.Name)
	cfg.Caught[pokemon.Name] = pokemon
	return nil
}

func displayLocations(data []byte, cfg *config) error {
	var result struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
		Next     string `json:"next"`
		Previous string `json:"previous"`
	}

	err := json.Unmarshal(data, &result)
	if err != nil {
		return err
	}

	cfg.Next = result.Next
	cfg.Previous = result.Previous
	cfg.Current = nil
	for _, location := range result.Results {
		cfg.Current = append(cfg.Current, location.Name)
	}

	for _, location := range cfg.Current {
		fmt.Println(location)
	}

	return nil
}

func displayPokemon(data []byte) error {
	var result struct {
		PokemonEncounters []struct {
			Pokemon struct {
				Name string `json:"name"`
			} `json:"pokemon"`
		} `json:"pokemon_encounters"`
	}

	err := json.Unmarshal(data, &result)
	if err != nil {
		return err
	}

	fmt.Println("Found Pokemon:")
	for _, encounter := range result.PokemonEncounters {
		fmt.Printf(" - %s\n", encounter.Pokemon.Name)
	}

	return nil
}

func printPokemonDetails(pokemon Pokemon) {
	fmt.Printf("Name: %s\n", pokemon.Name)
	fmt.Printf("Height: %d\n", pokemon.Height)
	fmt.Printf("Weight: %d\n", pokemon.Weight)
	fmt.Println("Stats:")
	for _, stat := range pokemon.Stats {
		fmt.Printf("  -%s: %d\n", stat.Stat.Name, stat.BaseStat)
	}
	fmt.Println("Types:")
	for _, typ := range pokemon.Types {
		fmt.Printf("  - %s\n", typ.Type.Name)
	}
}

func main() {
	cache := pokecache.NewCache(5 * time.Minute)
	cfg := &config{
		Cache:  cache,
		Caught: make(map[string]Pokemon),
	}

	commands := map[string]cliCommand{
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"map": {
			name:        "map",
			description: "Display the next 20 location areas",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "Display the previous 20 location areas",
			callback:    commandMapB,
		},
		"explore": {
			name:        "explore",
			description: "Explore a specific location area",
			callback:    commandExplore,
		},
		"catch": {
			name:        "catch",
			description: "Catch a specific Pokémon",
			callback:    commandCatch,
		},
		"inspect": {
			name:        "inspect",
			description: "Inspect a caught Pokémon",
			callback:    commandInspect,
		},
		"pokedex": {
			name:        "pokedex",
			description: "List all caught Pokémon",
			callback:    commandPokedex,
		},
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Pokedex > ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}
		cmdName := parts[0]
		args := parts[1:]
		if cmd, exists := commands[cmdName]; exists {
			cmd.callback(cfg, args)
		} else {
			fmt.Println("Unknown command:", input)
		}
	}
}
