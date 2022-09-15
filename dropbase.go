package main

import (
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func output(route string, res *http.Response) {
	switch res.StatusCode {
	case 200:
		log.Printf("successful revalidation on "+route+" with status: %s", res.Status)
	case 404:
		log.Printf("could not reach API on "+route+" with status: %s", res.Status)
	case 500:
		log.Printf("API on "+route+" has an internal server error with status: %s", res.Status)
	default:
		log.Printf("response did not return a 200, 404, or 500 status, status: %s", res.Status)
	}
}

func main() {
	app := pocketbase.New()

	routes := []string{}

	for i := 5; i < len(os.Args); i++ {
		routes = append(routes, os.Args[i])
	}

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("some error occured. err: %s", err)
	}

	app.OnRecordAfterUpdateRequest().Add(func(record *core.RecordUpdateEvent) error {
		values := url.Values{}
		values.Add("api_key", os.Getenv(("API_KEY")))
		values.Add("permalink", record.Record.GetStringDataValue("permalink"))

		for _, route := range routes {
			res, err := http.PostForm(route+"/api/revalidate", values)

			if err != nil {
				println(err)
			}

			defer res.Body.Close()

			output(route, res)
		}

		return nil
	})

	app.OnRecordAfterCreateRequest().Add(func(record *core.RecordCreateEvent) error {
		values := url.Values{}
		values.Add("api_key", os.Getenv(("API_KEY")))
		values.Add("permalink", record.Record.GetStringDataValue("permalink"))

		for _, route := range routes {
			res, err := http.PostForm(route+"/api/revalidate", values)

			if err != nil {
				println(err)
			}

			defer res.Body.Close()

			output(route, res)
		}

		return nil
	})

	app.OnRecordAfterDeleteRequest().Add(func(record *core.RecordDeleteEvent) error {
		values := url.Values{}
		values.Add("action", "delete")
		values.Add("api_key", os.Getenv(("API_KEY")))
		values.Add("permalink", record.Record.GetStringDataValue("permalink"))

		for _, route := range routes {
			res, err := http.PostForm(route+"/api/revalidate", values)

			if err != nil {
				println(err)
			}

			defer res.Body.Close()

			output(route, res)
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

}

func String(e *core.RecordUpdateEvent) {
	panic("unimplemented")
}