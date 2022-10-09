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
	"github.com/spf13/cobra"
)

// http response switcher for debugging
func response(route string, res *http.Response) {
	switch res.StatusCode {
	case 200:
		log.Printf("successful revalidation on "+route+", status: %s", res.Status)
	case 404:
		log.Printf("could not reach API on "+route+", status: %s, errror: %s", res.Status, res.Body)
	case 500:
		log.Printf("API on "+route+" has an internal server error, status: %s, errror: %s", res.Status, res.Body)
	default:
		log.Printf("response did not return a 200, 404, or 500 status, status: %s, errror: %s", res.Status, res.Body)
	}
}

// send API request with values to ISR routes
func send(routes []string, values url.Values) {
	for _, route := range routes {
		res, err := pester.PostForm(route+"/api/revalidate", values)

		if err != nil {
			log.Printf("error posting to "+route+", error: %s", err)
		}

		defer res.Body.Close()

		response(route, res)
	}
}

// get category from database
func getCategory(app *pocketbase.PocketBase, target string) string {
	collection, err := app.Dao().FindCollectionByNameOrId("categories")

	if err != nil {
		log.Printf("an error occured. err: %s", err)
	}

	result, err := app.Dao().FindFirstRecordByData(collection, "name", target)

	if err != nil {
		log.Print(err)
	}

	categoryPermalink := result.GetStringDataValue("permalink")

	return categoryPermalink
}

func main() {
	app := pocketbase.New()

	var routes []string

	err := godotenv.Load(".env")

	if err != nil {
		log.Printf("an error occured. err: %s", err)
	}

	var cached_product *models.Record
	var cached_product_categories []string
	var cached_category *models.Record

	app.OnRecordBeforeUpdateRequest().Add(func(e *core.RecordUpdateEvent) error {
		cached_product_categories = []string{}
		cached_category = nil
		cached_product = nil

		// cache product & it's categories if the update request collection is products
		if e.Record.Collection().Name == "products" {
			product, err := app.Dao().FindRecordById(e.Record.Collection(), e.Record.Id, nil)

			if err != nil {
				log.Printf("an error occured. err: %s", err)
			}

			cached_product = product

			for _, category := range e.Record.GetStringSliceDataValue("category") {
				cached_product_categories = append(cached_product_categories, getCategory(app, category))
			}
		}

		// cache category if the update request collection is categories
		if e.Record.Collection().Name == "categories" {
			category, err := app.Dao().FindRecordById(e.Record.Collection(), e.Record.Id, nil)

			if err != nil {
				log.Printf("an error occured. err: %s", err)
			}

			cached_category = category
		}

		return nil
	})

	app.OnRecordAfterUpdateRequest().Add(func(record *core.RecordUpdateEvent) error {
		values := url.Values{}
		values.Add("api_key", os.Getenv(("API_KEY")))

		// if the update request collection is products
		if record.Record.Collection().Name == "products" {
			var product_categories = []string{}

			for _, category := range record.Record.GetStringSliceDataValue("category") {
				product_categories = append(product_categories, getCategory(app, category))
			}

			// send new & old product permalinks & their categories
			values.Add("new_product", record.Record.GetStringDataValue("permalink"))
			values.Add("new_categories", strings.Join(product_categories, ","))

			values.Add("old_product", cached_product.GetStringDataValue("permalink"))
			values.Add("old_categories", strings.Join(cached_product_categories, ","))

			send(routes, values)
		}

		// if the update request collection is categories
		if record.Record.Collection().Name == "categories" {
			// send new & old category permalinks & their categories
			values.Add("old_category", cached_category.GetStringDataValue("permalink"))
			values.Add("new_category", record.Record.GetStringDataValue("permalink"))

			send(routes, values)
		}

		return nil
	})

	app.OnRecordAfterCreateRequest().Add(func(record *core.RecordCreateEvent) error {
		values := url.Values{}
		values.Add("api_key", os.Getenv(("API_KEY")))

		// if the creation request collection is products
		if record.Record.Collection().Name == "products" {
			var product_categories = []string{}

			for _, category := range record.Record.GetStringSliceDataValue("category") {
				product_categories = append(product_categories, getCategory(app, category))
			}

			// send newly created product permalink & it's categories
			values.Add("new_product", record.Record.GetStringDataValue("permalink"))
			values.Add("new_categories", strings.Join(product_categories, ","))

			send(routes, values)
		}

		// if the creation request collection is categories
		if record.Record.Collection().Name == "categories" {
			// send newly created category permalink
			values.Add("new_category", record.Record.GetStringDataValue("permalink"))

			send(routes, values)
		}

		return nil
	})

	app.OnRecordBeforeDeleteRequest().Add(func(record *core.RecordDeleteEvent) error {
		cached_product_categories = []string{}
		cached_category = nil
		cached_product = nil

		// cache product & it's categories if the delete request collection is products
		if record.Record.Collection().Name == "products" {
			product, err := app.Dao().FindRecordById(record.Record.Collection(), record.Record.Id, nil)

			if err != nil {
				log.Printf("an error occured. err: %s", err)
			}

			cached_product = product

			for _, category := range record.Record.GetStringSliceDataValue("category") {
				cached_product_categories = append(cached_product_categories, getCategory(app, category))
			}
		}

		// cache category if the delete request collection is categories
		if record.Record.Collection().Name == "categories" {
			category, err := app.Dao().FindRecordById(record.Record.Collection(), record.Record.Id, nil)

			if err != nil {
				log.Printf("an error occured. err: %s", err)
			}

			cached_category = category
		}

		return nil
	})

	app.OnRecordAfterDeleteRequest().Add(func(record *core.RecordDeleteEvent) error {
		values := url.Values{}
		values.Add("api_key", os.Getenv(("API_KEY")))

		// if the delete request collection is products
		if record.Record.Collection().Name == "products" {
			// send old deleted product permalink & it's categories
			values.Add("old_product", cached_product.GetStringDataValue("permalink"))
			values.Add("old_categories", strings.Join(cached_product_categories, ","))

			send(routes, values)
		}

		// if the delete request collection is cateogries
		if record.Record.Collection().Name == "categories" {
			// send old deleted category permalink
			values.Add("old_category", cached_category.GetStringDataValue("permalink"))

			send(routes, values)
		}

		return nil
	})

	// create a custom start command
	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "a custom start command to target ISR routes using flags",
		Run: func(command *cobra.Command, args []string) {
			server, _ := command.Flags().GetString("http")
			ISR_Routes, _ := command.Flags().GetStringSlice("routes")
			routes = append(routes, ISR_Routes...)

			app.RootCmd.SetArgs([]string{"serve", "--http", server, "--encryptionEnv=" + os.Getenv("ENCRYPTION_KEY")})

			if err := app.Start(); err != nil {
				log.Fatal(err)
			}
		},
	}

	app.RootCmd.AddCommand(startCmd)

	// set flags for start command
	startCmd.Flags().String("http", "0.0.0.0:8090", "custom http address & port")
	startCmd.Flags().StringSlice("routes", []string{}, "routes for incremental static regeneration")
	startCmd.MarkFlagRequired("routes")

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

}
