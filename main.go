package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/pkg/errors"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

const (
	urlPefix = "https://my.uscis.gov/appointmentscheduler-appointment/field-offices/zipcode/"
)

var (
	// SF, San Jose, Oakland
	// zipCodes = []string{"94016", "94088", "94501"}
	zipCodes = []string{"94016"}
)

type response struct {
	Description string        `json:"description"`
	TimeSlots   []interface{} `json:"timeSlots"`
}

func main() {
	poll()

	ticker := time.NewTicker(2 * time.Hour)
	for {
		select {
		case <-ticker.C:
			poll()
		}
	}
}

func poll() {
	fmt.Println("Polling...")

	for _, zipCode := range zipCodes {
		location, err := findAvailabiltiy(zipCode)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			continue
		}

		// if location == "" {
		// 	fmt.Printf("Cannot find location for zipcode %s\n", zipCode)
		// 	continue
		// }

		content := fmt.Sprintf(
			"Found a slot in %s, visit https://my.uscis.gov/appointmentscheduler-appointment/ca/en/office-search and search for %s",
			location,
			zipCode,
		)

		if err := sendMsg(content); err != nil {
			fmt.Printf("Error: %s\n", err)
		}
	}
}

func sendMsg(content string) error {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")

	fromNumber := os.Getenv("TWILIO_FROM_NUMBER")
	toNumebr := os.Getenv("TWILIO_TO_NUMBER")

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	params := (&twilioApi.CreateMessageParams{}).
		SetTo(toNumebr).
		SetFrom(fromNumber).
		SetBody(content)

	resp, err := client.Api.CreateMessage(params)
	if err != nil {
		return errors.Wrap(err, "Error sending SMS messsage")
	}

	response, _ := json.Marshal(*resp)
	fmt.Println("Response: " + string(response))

	return nil
}

func findAvailabiltiy(zipcode string) (string, error) {
	url := urlPefix + zipcode

	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "Failed to query USCIS")
	}

	defer resp.Body.Close()

	var data []response
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", errors.Wrap(err, "Failed to decode response from USCIS")
	}

	for _, d := range data {
		if len(d.TimeSlots) != 0 {
			return d.Description, nil
		}
	}

	return "", nil
}
