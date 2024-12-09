package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/robinsoj/pokedex/internal/pokecache"
	"github.com/robinsoj/pokedex/internal/pokestructs"
)

type cliCommand struct {
	name        string
	description string
	callback    func([]string) error
}

var return_map map[string]cliCommand
var pokemon_collection map[string]pokestructs.PokemonCreature
var fwd_link string
var back_link string
var cache *pokecache.Cache

func init() {
	pokemon_collection = make(map[string]pokestructs.PokemonCreature)
	return_map = map[string]cliCommand{
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
			description: "Displays the next 20 map locations",
			callback:    mapForward,
		},
		"mapb": {
			name:        "mapb",
			description: "Displays the previous 20 map locations",
			callback:    mapBack,
		},
		"explore": {
			name:        "explore",
			description: "Displays the Pokemon in a specified zone",
			callback:    exploreZone,
		},
		"catch": {
			name:        "catch",
			description: "Catch the specified Pokemon",
			callback:    catchPokemon,
		},
		"inspect": {
			name:        "inspect",
			description: "View information about a specified Pokemon",
			callback:    inspectPokemon,
		},
		"pokedex": {
			name:        "pokedex",
			description: "View the entire contents of your Pokedex",
			callback:    printPokedex,
		},
	}
}

func commandHelp(params []string) error {
	fmt.Println()
	fmt.Println("Welcome to PokeDex!")
	fmt.Println("Usage:")
	fmt.Println()
	for _, cmd := range return_map {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	fmt.Println()
	return nil
}

func commandExit(params []string) error {
	fmt.Println("Goodbye")
	os.Exit(0)
	return nil
}

func mapForward(params []string) error {
	if fwd_link == "" {
		fwd_link = "https://pokeapi.co/api/v2/location-area/"
	}
	data, err := downloadPage(fwd_link)
	if err != nil {
		return err
	}
	var pokeloc pokestructs.PokeLocation
	if err := json.Unmarshal(data, &pokeloc); err != nil {
		return err
	}
	fwd_link = decodeUrl(pokeloc.Next)
	back_link = decodeUrl(pokeloc.Previous)
	print_areas(pokeloc)
	return nil
}

func mapBack(params []string) error {
	if back_link == "" {
		return errors.New("trying to navigate to before the beginning of the list")
	}
	data, err := downloadPage(back_link)
	if err != nil {
		return err
	}
	var pokeloc pokestructs.PokeLocation
	if err := json.Unmarshal(data, &pokeloc); err != nil {
		return err
	}
	fwd_link = decodeUrl(pokeloc.Next)
	back_link = decodeUrl(pokeloc.Previous)
	print_areas(pokeloc)
	return nil
}

func exploreZone(zones []string) error {
	if zones == nil {
		return errors.New("no zone passed to exploreZone")
	}
	fullUrl := "https://pokeapi.co/api/v2/location-area/" + zones[0]
	data, err := downloadPage(fullUrl)
	if err != nil {
		return err
	}
	var pokeEnc pokestructs.PokeEncounters
	if err := json.Unmarshal(data, &pokeEnc); err != nil {
		return err
	}
	fmt.Println("Exploring", zones[0], "...")
	print_encounters(pokeEnc)
	return nil
}

func catchPokemon(pokemon []string) error {
	if pokemon == nil {
		return errors.New("no pokemon passed to catchPokemon")
	}
	fullUrl := "https://pokeapi.co/api/v2/pokemon/" + pokemon[0]
	data, err := downloadPage(fullUrl)
	if err != nil {
		return err
	}
	var pokeCreature pokestructs.PokemonCreature
	if err := json.Unmarshal(data, &pokeCreature); err != nil {
		return err
	}
	fmt.Println("Throwing a Pokeball at", pokemon[0], "...")
	maxPossibleXp := 609
	curBaseXP := pokeCreature.BaseExperience
	chance := float64((maxPossibleXp - curBaseXP)) / float64(maxPossibleXp)
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	attempt := r.Float64()
	if chance > attempt {
		pokemon_collection[pokemon[0]] = pokeCreature
		fmt.Println(pokemon[0], "was caught!")
	} else {
		fmt.Println(pokemon[0], "escaped!")
	}
	return nil
}

func inspectPokemon(pokemon []string) error {
	data, ok := pokemon_collection[pokemon[0]]
	if !ok {
		fmt.Println("you have not caught that pokemon")
		return nil
	}
	fmt.Println("Name:", data.Name)
	fmt.Println("Height:", data.Height)
	fmt.Println("Weight:", data.Weight)
	fmt.Println("Stats:")
	for _, stat := range data.Stats {
		fmt.Println(" -", stat.Stat.Name, ":", stat.BaseStat)
	}
	fmt.Println("Types:")
	for _, kind := range data.Types {
		fmt.Println(" -", kind.Type.Name)
	}
	return nil
}

func downloadPage(url string) ([]byte, error) {
	value, ok := cache.Get(url)
	if ok {
		return value, nil
	}
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	cache.Add(url, body)
	return body, nil
}

func printPokedex(_ []string) error {
	fmt.Println("Your Pokedex:")
	for key := range pokemon_collection {
		fmt.Println(" -", key)
	}
	return nil
}

func decodeUrl(url string) string {
	var ret_str string
	if url == "null" {
		ret_str = ""
	} else {
		ret_str = url
	}
	return ret_str
}

func print_areas(location pokestructs.PokeLocation) error {
	if location.Results == nil {
		return errors.New("trying to print empty location structure")
	}
	for _, result := range location.Results {
		fmt.Println(result.Name)
	}
	return nil
}

func print_encounters(encounter pokestructs.PokeEncounters) error {
	if encounter.PokemonEncounters == nil {
		return errors.New("trying to print empty encounter structure")
	}
	fmt.Println("Found Pokemon")
	for _, pokemon := range encounter.PokemonEncounters {
		fmt.Println(" - ", pokemon.Pokemon.Name)
	}
	return nil
}

func main() {
	fwd_link = ""
	back_link = ""
	cache = pokecache.NewCache(5 * time.Second)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("pokedex > ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		words := strings.Split(input, " ")
		if cmd, exists := return_map[words[0]]; exists {
			if err := cmd.callback(words[1:]); err != nil {
				fmt.Println("Error: ", err)
			}
		} else {
			fmt.Println("Invalid selection")
		}
	}
}
