package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	htf "healthplanet-to-fitbit"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	godotenv.Load(".env")

	healthPlanetAccessToken := os.Getenv("HEALTHPLANET_ACCESS_TOKEN")
	fitbitClientId := os.Getenv("FITBIT_CLIENT_ID")
	fitbitClientSecret := os.Getenv("FITBIT_CLIENT_SECRET")
	// fitbitAccessToken := os.Getenv("FITBIT_ACCESS_TOKEN")
	fitbitRefreshToken := os.Getenv("FITBIT_REFRESH_TOKEN")

	fitbitAccessToken, err := refreshAccessToken(fitbitClientId, fitbitRefreshToken)
	fmt.Println(err)

	// Initialize API clients
	healthPlanetAPI := htf.HealthPlanetAPI{
		AccessToken: healthPlanetAccessToken,
	}
	fitbitApi := htf.NewFitbitAPI(fitbitClientId, fitbitClientSecret, fitbitAccessToken, fitbitRefreshToken)

	// Initialize Context
	ctx := context.Background()

	// Get data from HealthPlanet
	scanData, err := healthPlanetAPI.AggregateInnerScanData(ctx)
	if err != nil {
		log.Fatalf("failed to aggregate inner scan data: %+v", err)
	}

	// Save data to Fitbit
	for t, data := range scanData {
		weightLog, err := fitbitApi.GetBodyWeightLog(t)
		if err != nil {
			log.Fatalf("failed to get weight log from fitbit: %+v", err)
		}

		if len(weightLog.Weight) > 0 {
			log.Printf("%s: record is found", t)
			continue
		}

		if data.Weight != nil {
			if err := fitbitApi.CreateWeightLog(*data.Weight, t); err != nil {
				log.Fatalf("failed to create weight log: time: %s, err: %+v", t, err)
			}
		}

		if data.Fat != nil {
			if err := fitbitApi.CreateBodyFatLog(*data.Fat, t); err != nil {
				log.Fatalf("failed to create fat log: time: %s, err: %+v", t, err)
			}
		}

		printFloat := func(f *float64) string {
			if f == nil {
				return "nil"
			}
			return fmt.Sprintf("%.2f", *f)
		}

		log.Printf("%s: saved, weight: %s, fat: %s", t, printFloat(data.Weight), printFloat(data.Fat))
	}

	log.Printf("done")
}

func refreshAccessToken(clientID string, currentRefreshToken string) (string, error) {
	endpoint := "https://api.fitbit.com/oauth2/token"

	data := url.Values{}
	data.Add("grant_type", "refresh_token")
	data.Add("client_id", clientID)
	data.Add("refresh_token", currentRefreshToken)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error in response: %v", err)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	resData := make(map[string]interface{})
	if err := dec.Decode(&resData); err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}

	accessToken := resData["access_token"].(string)
	newRefreshToken := resData["refresh_token"].(string)

	updateEnvFile(accessToken, newRefreshToken)
	fmt.Println("Access token and refresh token updated.")
	return accessToken, nil
}

func updateEnvFile(accessToken, refreshToken string) {
	envFilePath := ".env"
	envFile, err := os.OpenFile(envFilePath, os.O_RDWR, 0644)
	if err != nil {
		fmt.Printf("Error opening .env file: %v\n", err)
		return
	}
	defer envFile.Close()

	// Read the existing content
	fileInfo, err := envFile.Stat()
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		return
	}

	fileSize := fileInfo.Size()
	buffer := make([]byte, fileSize)
	_, err = envFile.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Split the content by lines
	lines := bytes.Split(buffer, []byte("\n"))

	// Update the lines containing FITBIT_ACCESS_TOKEN and FITBIT_REFRESH_TOKEN
	for i, line := range lines {
		if bytes.HasPrefix(line, []byte("FITBIT_ACCESS_TOKEN=")) {
			lines[i] = []byte(fmt.Sprintf("FITBIT_ACCESS_TOKEN=%s", accessToken))
		}
		if bytes.HasPrefix(line, []byte("FITBIT_REFRESH_TOKEN=")) {
			lines[i] = []byte(fmt.Sprintf("FITBIT_REFRESH_TOKEN=%s", refreshToken))
		}
	}

	// Write the updated content back to the file
	envFile.Seek(0, 0)
	envFile.Truncate(0)
	_, err = envFile.Write(bytes.Join(lines, []byte("\n")))
	if err != nil {
		fmt.Printf("Error writing to .env file: %v\n", err)
	}
}
