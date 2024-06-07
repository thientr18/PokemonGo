package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type Pokemon struct {
	Id       string   `json:"ID"`
	Name     string   `json:"Name"`
	Types    []string `json:"types"`
	Link     string   `json:"URL"`
	PokeInfo PokeInfo `json:"Poke-Information"`
}
type PokeInfo struct {
	Hp    int      `json:"HP"`
	Atk   int      `json:"ATK"`
	Def   int      `json:"DEF"`
	SpAtk int 		`json:"Sp.Atk"`
	SpDef int		`json:"Sp.Def"`
	Speed int		`json:"Speed"`
}

func main() {

	fmt.Println("Connecting to the web")

	resp, err := http.Get("https://pokemondb.net/pokedex/national")
	if err != nil {
		fmt.Println("Error fetching Poke Homepage: ", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body: ", err)
		return
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		fmt.Println("Error parsing HTML: ", err)
		return
	}
	fmt.Println("Downloading... ")
	// genres := getGenres(doc)
	// var allGenre []Genres
	// var allBook []Poke
	pokemons := getPokedex(doc)
	var allPoke []Pokemon
	var allPokeInfo PokeInfo

	for _, poke := range pokemons {

		allPokeInfo = getDetail(poke.Link)
		poke.PokeInfo = allPokeInfo
		allPoke = append(allPoke, poke)
		allPokeInfo = PokeInfo{}
	}

	jsonData, err := json.MarshalIndent(allPoke, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling to JSON: ", err)
		return
	}

	err = ioutil.WriteFile("pokedex.json", jsonData, 0644)
	if err != nil {
		fmt.Println("Error writing JSON to file: ", err)
		return
	}

	fmt.Println("Pokedex data has been written to pokedex.json")
}

func getPokedex(n *html.Node) []Pokemon {
	var pokemon []Pokemon
	var currentPoke Pokemon
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && attr.Val == "infocard-lg-data text-muted" {
					currentPoke.Id = strings.Split(getOnce(n, "small"), "a")[0]
					currentPoke.Name = getInsideTag(n, "a", "class", "ent-name")
					links := getStringElement(n, "a", "href")
					currentPoke.Link = strings.Split(links, "/type")[0]
					types := getInsideTag(n, "a", "class", "itype")
					typeSplit := strings.Split(types, " ")
					for _, eachType := range typeSplit {
						if eachType != "" {
							currentPoke.Types = append(currentPoke.Types, eachType)
						}
					}
					if currentPoke.Name != "" {
						pokemon = append(pokemon, currentPoke)
						currentPoke = Pokemon{}
					}

				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return pokemon
}

func getDetail(url string) PokeInfo {
	resp, err := http.Get("https://pokemondb.net" + url)
	if err != nil {
		fmt.Println("Error fetching WEBTOON homepage: ", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body: ", err)
		os.Exit(1)
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		fmt.Println("Error parsing HTML: ", err)
		os.Exit(1)
	}
	var pokeInfo PokeInfo
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && attr.Val == "vitals-table" {
					result := getOnce(n, "th")
					stats := strings.Split(result, " ")
					for _, st := range stats {
						if st == "HP"{
							listStat := getStatNumber(n, "td", "class", "cell-num")
							for i:= 0; i < len(listStat); i++{
								pokeInfo.Hp = listStat[0]
								pokeInfo.Atk = listStat[3]
								pokeInfo.Def = listStat[6]
								pokeInfo.SpAtk = listStat[9]
								pokeInfo.SpDef = listStat[12]
								pokeInfo.Speed = listStat[15]
								
							}
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return pokeInfo
}

func getInsideTag(n *html.Node, data, key, val string) string {
	var result = ""
	if n.Type == html.ElementNode && n.Data == data {
		for _, attr := range n.Attr {
			if attr.Key == key && strings.Split(attr.Val, " ")[0] == val {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					result += " " + strings.TrimSpace(c.Data)
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += getInsideTag(c, data, key, val)
	}
	return result
}

func getOnce(n *html.Node, tagName string) string {
	var result string
	if n.Type == html.ElementNode && n.Data == tagName {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			result += " " +strings.TrimSpace(c.Data)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += getOnce(c, tagName) 
	}
	return result
}
func getStringElement(n *html.Node, data, key string) string {
	var result string
	if n.Type == html.ElementNode && n.Data == data {
		for _, attr := range n.Attr {
			if attr.Key == key {
				result = attr.Val
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += getStringElement(c, data, key)
	}
	return result
}
func getStatNumber(n *html.Node, tagName, key, val string) []int {
	var numbers []int
	if n.Type == html.ElementNode && n.Data == tagName {
		for _, attr := range n.Attr {
			if attr.Key == key && attr.Val == val {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					number, _:= strconv.Atoi(strings.TrimSpace(c.Data))
					numbers= append(numbers, number)
				}
			}
		}
	}
	for c:= n.FirstChild; c!= nil; c = c.NextSibling{
		number :=getStatNumber(c, tagName, key, val)
		for _, c:= range number{
			numbers = append(numbers, c)
		}
	}
	return numbers
}