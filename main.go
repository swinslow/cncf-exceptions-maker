// SPDX-License-Identifier: Apache-2.0
// Copyright The Linux Foundation

package main

// based on quickstart from https://developers.google.com/sheets/api/quickstart/go
// and code from https://github.com/gsuitedevs/go-samples/blob/master/sheets/quickstart/quickstart.go
// with the following copyright and license notice:
//
// Copyright Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spdx/tools-golang/v0/tvsaver"
	"github.com/swinslow/cncf-exceptions-maker/pkg/exceptionmaker"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// ExceptionConfig holds the configuration options for the exception maker.
type ExceptionConfig struct {
	SpreadsheetID string `json:"spreadsheetId"`
}

func loadConfig(filename string) (*ExceptionConfig, error) {
	js, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", filename, err)
	}

	cfg := ExceptionConfig{}
	err = json.Unmarshal(js, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling from JSON: %v", err)
	}

	return &cfg, nil
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Unable to get user home directory: %v", err)
	}

	b, err := ioutil.ReadFile(filepath.Join(home, ".google-sheets-cncf-exceptions-credentials.json"))
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	cfg, err := loadConfig(filepath.Join(home, ".cncf-exceptions-config"))
	readRange := "Approved!A2:I"
	resp, err := srv.Spreadsheets.Values.Get(cfg.SpreadsheetID, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	doc := exceptionmaker.MakeDocument()

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
	} else {
		rowNum := 2
		for _, row := range resp.Values {
			// check whether this row is complete
			if len(row) < 9 {
				log.Printf("==> INCOMPLETE ROW (%d): %v\n", len(row), row)
				continue
			}

			pkg, err := exceptionmaker.MakePackageFromRow(row, rowNum)
			if err != nil {
				log.Fatalf("Unable to convert rowNum %d data to SPDX package: %v\n", rowNum, err)
			}
			doc.Packages = append(doc.Packages, pkg)

			rowNum++
		}
	}

	// and write to disk
	fileOut := fmt.Sprintf("cncf-exceptions-%s.spdx", time.Now().Format("2006-01-02"))
	w, err := os.Create(fileOut)
	if err != nil {
		log.Fatalf("Error while opening %v for writing: %v", fileOut, err)
	}
	defer w.Close()

	err = tvsaver.Save2_1(doc, w)
	if err != nil {
		log.Fatalf("Error while saving %v: %v", fileOut, err)
	}

	fmt.Printf("Saved exceptions list as SPDX to %s\n", fileOut)

	subsets := exceptionmaker.ConvertSPDXToJSONPackageSubset(doc)
	jsonStr, err := json.MarshalIndent(subsets, "", "  ")
	if err != nil {
		log.Fatalf("Error while marshalling to JSON: %v", err)
	}

	jsonOut := fmt.Sprintf("cncf-exceptions-%s.json", time.Now().Format("2006-01-02"))
	j, err := os.Create(jsonOut)
	if err != nil {
		log.Fatalf("Error while opening %v for writing: %v", fileOut, err)
	}
	defer j.Close()

	_, err = j.Write(jsonStr)
	if err != nil {
		log.Fatalf("Error while saving %v: %v", jsonOut, err)
	}

	fmt.Printf("Saved exceptions list as JSON to %s\n", jsonOut)
}
