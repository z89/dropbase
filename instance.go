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

func main() {
	routes := []string{"http://localhost:3000/"}

	app := pocketbase.New()

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("some error occured. err: %s", err)
	}

	app.OnRecordAfterUpdateRequest().Add(func(record *core.RecordUpdateEvent) error {
		values := url.Values{}
		values.Add("api_key", os.Getenv(("API_KEY")))
		values.Add("permalink", record.Record.GetStringDataValue("permalink"))

		for _, route := range routes {
			res, err := http.PostForm(route+"api/revalidate", values)

			if err != nil {
				println(err)
			}

			defer res.Body.Close()

			log.Printf("record has been updated, status: %s", res.Status)
		}

		return nil
	})

	app.OnRecordAfterCreateRequest().Add(func(record *core.RecordCreateEvent) error {
		values := url.Values{}
		values.Add("api_key", os.Getenv(("API_KEY")))
		values.Add("permalink", record.Record.GetStringDataValue("permalink"))

		for _, route := range routes {
			res, err := http.PostForm(route+"api/revalidate", values)

			if err != nil {
				println(err)
			}

			defer res.Body.Close()

			log.Printf("record has been created, status: %s", res.Status)
		}

		return nil
	})

	app.OnRecordAfterDeleteRequest().Add(func(record *core.RecordDeleteEvent) error {
		values := url.Values{}
		values.Add("action", "delete")
		values.Add("api_key", os.Getenv(("API_KEY")))
		values.Add("permalink", record.Record.GetStringDataValue("permalink"))

		for _, route := range routes {
			res, err := http.PostForm(route+"api/revalidate", values)

			if err != nil {
				println(err)
			}

			defer res.Body.Close()

			log.Printf("record has been deleted, status: %s", res.Status)
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
