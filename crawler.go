package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var counter int = 0
var ingred map[string]string

type Recipe struct {
	Instructions string
	Ingredients  []string //[]map[string]string
	Name         string
	Stats        []string
}

func main() {
	//Mongo
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://user1:dogdogdog333@cluster0.xojje.gcp.mongodb.net/recipesDB?retryWrites=true&w=majority"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 3600*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	col := client.Database("recipesDB").Collection("recipes")
	defer client.Disconnect(ctx)

	//Colly collectors
	c := colly.NewCollector(
		// Visit only domains: hackerspaces.org, wiki.hackerspaces.org
		colly.AllowedDomains("recepti.gotvach.bg"),
		colly.CacheDir("./gotvach_cache"),
	)
	statsCollector := c.Clone()
	recipeCollector := c.Clone()
	//recipes := make([]Recipe, 0)
	//global scraper
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if e.Attr("class") == "title" {
			link := e.Attr("href")
			recipeCollector.Visit(link)
			log.Println("Searching", link)
		}
		if e.Attr("class") == "prev" {
			e.Request.Visit(e.Attr("href"))
		}

	})
	//Print Visiting on Request
	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting ", r.URL.String())
	})
	var stats []string
	recipeCollector.OnHTML("div[id = wrap]", func(e *colly.HTMLElement) {
		fmt.Printf("Recipe found ")
		var ingredients []string
		var instructions []string
		title := e.ChildText("div[id = content] > div[id=recEntity] > .combocolumn > h1")
		if title == "" {
			title = "No title Found"
			log.Println("No title found")
		}
		recipe := Recipe{
			Name: title,
		}

		e.ForEach("section.products > ul > li", func(_ int, el *colly.HTMLElement) {
			ingredients = append(ingredients, el.Text)
			//ingredient := string([]rune(el.Text)[0,strings.Index(el.Text," -"))
			//ingred[ingredient]
		})
		recipe.Ingredients = ingredients
		e.ForEach("div.text", func(_ int, el *colly.HTMLElement) {
			instructions = append(instructions, el.ChildText("p.desc"))
		})
		recipe.Instructions = strings.Join(instructions, " ")
		statsCollector.Visit(fmt.Sprintf("%s?=1", e.Request.URL.String()))
		recipe.Stats = stats
		stats = nil
		_, insertErr := col.InsertOne(ctx, recipe)
		if insertErr != nil {
			fmt.Println("InsertOne ERROR:", insertErr)
			os.Exit(1) // safely exit script on error
		}
	})
	statsCollector.OnHTML("div[id = wrap]", func(e *colly.HTMLElement) {
		e.ForEach("div[id = content] > div[id = recEntity] > .combocolumn > .stickbox > .maincolumn > div[id=recContent] > .stats > .bottom > ul > li", func(_ int, el *colly.HTMLElement) {
			stats = append(stats, el.Text)
		})
	})
	c.Visit("https://recepti.gotvach.bg/2000")
}
