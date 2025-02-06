package htf

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type GetWeightLogResponse struct {
	Weight []struct {
		BMI    float64 `json:"bmi"`
		Date   string  `json:"date"`
		Fat    float64 `json:"fat"`
		LogId  int64   `json:"logId"`
		Source string  `json:"source"`
		Time   string  `json:"time"`
		Weight float64 `json:"weight"`
	} `json:"weight"`
}

func GetFitbitConfig(clientID string, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"weight"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://www.fitbit.com/oauth2/authorize",
			TokenURL: "https://api.fitbit.com/oauth2/token",
		},
		RedirectURL: "http://localhost:8080/callback",
	}
}

type FitbitAPI struct {
	Client *http.Client
	TokenSource oauth2.TokenSource
}

func NewFitbitAPI(clientID string, clientSecret string, accessToken string, refreshToken string) *FitbitAPI {
	cfg := GetFitbitConfig(clientID, clientSecret)
	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	tokenSource := cfg.TokenSource(context.Background(), token)
	// cli := cfg.Client(context.Background(), token)
	cli := oauth2.NewClient(context.Background(), tokenSource)
	return &FitbitAPI{
		Client: cli,
		TokenSource: tokenSource,
	}
}

func (api *FitbitAPI) CreateWeightLog(weight float64, date time.Time) error {
	values := url.Values{}
	values.Add("weight", strconv.FormatFloat(weight, 'f', 2, 64))
	values.Add("date", date.Format("2006-01-02"))
	values.Add("time", date.Format("15:04:05"))

	res, err := api.Client.Post(fmt.Sprintf("https://api.fitbit.com/1/user/-/body/log/weight.json?%s", values.Encode()), "application/json", nil)
	if err != nil {
		return errors.Wrap(err, "failed to create weight log in fitbit")
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || 400 <= res.StatusCode {
		return errors.Errorf("failed to create weight log in fitbit(invalid status code): %d", res.StatusCode)
	}

	return nil
}

func (api *FitbitAPI) CreateBodyFatLog(fat float64, date time.Time) error {
	values := url.Values{}
	values.Add("fat", strconv.FormatFloat(fat, 'f', 2, 64))
	values.Add("date", date.Format("2006-01-02"))
	values.Add("time", date.Format("15:04:05"))

	res, err := api.Client.Post(fmt.Sprintf("https://api.fitbit.com/1/user/-/body/log/fat.json?%s", values.Encode()), "application/json", nil)
	if err != nil {
		return errors.Wrap(err, "failed to create fat log in fitbit")
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || 400 <= res.StatusCode {
		return errors.Errorf("failed to create fat log in fitbit(invalid status code): %d", res.StatusCode)
	}

	return nil
}

func (api *FitbitAPI) GetBodyWeightLog(date time.Time) (*GetWeightLogResponse, error) {
	token, err := api.TokenSource.Token()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token")
	}
	saveToken(token)

	formattedDate := date.Format("2006-01-02")

	res, err := api.Client.Get(fmt.Sprintf("https://api.fitbit.com/1/user/-/body/log/weight/date/%s.json", formattedDate))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get weight log in fitbit")
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || 400 <= res.StatusCode {
		return nil, errors.Errorf("failed to get weight log in fitbit(invalid status code): %d", res.StatusCode)
	}

	dec := json.NewDecoder(res.Body)
	var resData GetWeightLogResponse
	if err := dec.Decode(&resData); err != nil {
		return nil, errors.Wrap(err, "failed to parse weight log in fitbit")
	}

	return &resData, nil
}


func updateEnvFile(key, value string) error {
	file, err := os.Open(".env")
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, key+"=") {
			lines = append(lines, key+"="+value)
		} else {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	file, err = os.Create(".env")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}
	return writer.Flush()
}

func saveToken(token *oauth2.Token) {
	// fmt.Printf("AccessToken: %s\n", token.AccessToken)
	// fmt.Printf("RefreshToken: %s\n", token.RefreshToken)
	// update the token in the .env file
	// Assuming you have a function to update the .env file
	err := updateEnvFile("ACCESS_TOKEN", token.AccessToken)
	if err != nil {
		fmt.Printf("failed to update access token: %v\n", err)
	}
	err = updateEnvFile("REFRESH_TOKEN", token.RefreshToken)
	if err != nil {
		fmt.Printf("failed to update refresh token: %v\n", err)
	}
	// print to console
	fmt.Printf("AccessToken: %s\n", token.AccessToken)
	fmt.Printf("RefreshToken: %s\n", token.RefreshToken)
}
