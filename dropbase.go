package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/sethgrid/pester"
)

func response(route string, res *http.Response) {
	switch res.StatusCode {
	case 200:
		log.Printf("successful revalidation on "+route+", status: %s", res.Status)
	case 404:
		log.Printf("could not reach API on "+route+", status: %s", res.Status)
	case 500:
		log.Printf("API on "+route+" has an internal server error, status: %s", res.Status)
	default:
		log.Printf("response did not return a 200, 404, or 500 status, status: %s", res.Status)
	}
}

func send(routes []string, values url.Values) {
	for _, route := range routes {
		res, err := pester.PostForm(route+"/api/revalidate", values)

		if err != nil {
			log.Printf("error posting to "+route+": %s", err)
		}

		defer res.Body.Close()

		response(route, res)
	}
}

func getCategory(app *pocketbase.PocketBase, target string) string {
	collection, err := app.Dao().FindCollectionByNameOrId("categories")

	if err != nil {
		log.Printf("some error occured. err: %s", err)
	}

	result, err := app.Dao().FindFirstRecordByData(collection, "name", target)

	if err != nil {
		log.Print(err)
	}

	category := result.GetStringDataValue("permalink")

	return category
}

func main() {
	app := pocketbase.New()

	var routes []string

	for i := 5; i < len(os.Args); i++ {
		routes = append(routes, os.Args[i])
	}

	err := godotenv.Load(".env")

	if err != nil {
		log.Printf("some error occured. err: %s", err)
	}

	var cached_product *models.Record
	var cached_product_categories []string
	var cached_category *models.Record

	app.OnRecordBeforeUpdateRequest().Add(func(e *core.RecordUpdateEvent) error {
		cached_product_categories = []string{}
		cached_category = nil
		cached_product = nil

		if e.Record.Collection().Name == "products" {
			product, err := app.Dao().FindRecordById(e.Record.Collection(), e.Record.Id, nil)

			if err != nil {
				log.Printf("some error occured. err: %s", err)
			}

			cached_product = product

			for _, category := range e.Record.GetStringSliceDataValue("category") {
				cached_product_categories = append(cached_product_categories, getCategory(app, category))
			}
		} else if e.Record.Collection().Name == "categories" {
			category, err := app.Dao().FindRecordById(e.Record.Collection(), e.Record.Id, nil)

			if err != nil {
				log.Printf("some error occured. err: %s", err)
			}

			cached_category = category
		}

		return nil
	})

	app.OnRecordAfterUpdateRequest().Add(func(record *core.RecordUpdateEvent) error {
		values := url.Values{}
		values.Add("api_key", os.Getenv(("API_KEY")))

		if record.Record.Collection().Name == "categories" {
			values.Add("old_category", cached_category.GetStringDataValue("permalink"))
			values.Add("new_category", record.Record.GetStringDataValue("permalink"))

			send(routes, values)
		} else if record.Record.Collection().Name == "products" {
			var product_categories = []string{}

			for _, category := range record.Record.GetStringSliceDataValue("category") {
				product_categories = append(product_categories, getCategory(app, category))
			}

			values.Add("new_product", record.Record.GetStringDataValue("permalink"))
			values.Add("new_categories", strings.Join(product_categories, ","))

			values.Add("old_product", cached_product.GetStringDataValue("permalink"))
			values.Add("old_categories", strings.Join(cached_product_categories, ","))

			send(routes, values)
		}

		return nil
	})

	app.OnRecordAfterCreateRequest().Add(func(record *core.RecordCreateEvent) error {
		values := url.Values{}
		values.Add("api_key", os.Getenv(("API_KEY")))

		if record.Record.Collection().Name == "categories" {
			values.Add("new_category", record.Record.GetStringDataValue("permalink"))

			send(routes, values)
		} else if record.Record.Collection().Name == "products" {
			var product_categories = []string{}

			for _, category := range record.Record.GetStringSliceDataValue("category") {
				product_categories = append(product_categories, getCategory(app, category))
			}

			values.Add("new_product", record.Record.GetStringDataValue("permalink"))
			values.Add("new_categories", strings.Join(product_categories, ","))

			send(routes, values)
		}

		return nil
	})

	app.OnRecordBeforeDeleteRequest().Add(func(record *core.RecordDeleteEvent) error {
		cached_product_categories = []string{}
		cached_category = nil
		cached_product = nil

		if record.Record.Collection().Name == "products" {
			product, err := app.Dao().FindRecordById(record.Record.Collection(), record.Record.Id, nil)

			if err != nil {
				log.Printf("some error occured. err: %s", err)
			}

			cached_product = product

			for _, category := range record.Record.GetStringSliceDataValue("category") {
				cached_product_categories = append(cached_product_categories, getCategory(app, category))
			}
		} else if record.Record.Collection().Name == "categories" {
			category, err := app.Dao().FindRecordById(record.Record.Collection(), record.Record.Id, nil)

			if err != nil {
				log.Printf("some error occured. err: %s", err)
			}

			cached_category = category
		}

		return nil
	})

	app.OnRecordAfterDeleteRequest().Add(func(record *core.RecordDeleteEvent) error {
		values := url.Values{}
		values.Add("api_key", os.Getenv(("API_KEY")))

		if record.Record.Collection().Name == "categories" {
			values.Add("old_category", cached_category.GetStringDataValue("permalink"))

			send(routes, values)
		} else if record.Record.Collection().Name == "products" {
			values.Add("old_product", cached_product.GetStringDataValue("permalink"))
			values.Add("old_categories", strings.Join(cached_product_categories, ","))

			send(routes, values)
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

}
