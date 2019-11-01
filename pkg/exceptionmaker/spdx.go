// SPDX-License-Identifier: Apache-2.0
// Copyright The Linux Foundation

package exceptionmaker

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spdx/tools-golang/v0/spdx"
)

// MakeDocument creates an SPDX Document2_1 entry to which
// Packages will be added.
func MakeDocument() *spdx.Document2_1 {
	datestr := time.Now().Format("2006-01-02")
	ci := &spdx.CreationInfo2_1{
		SPDXVersion:          "SPDX-2.1",
		DataLicense:          "CC0-1.0",
		SPDXIdentifier:       "SPDXRef-DOCUMENT",
		DocumentName:         fmt.Sprintf("cncf-exceptions-%s", datestr),
		DocumentNamespace:    fmt.Sprintf("https://github.com/cncf/foundation/license-exceptions-%s", datestr),
		CreatorOrganizations: []string{"CNCF"},
		CreatorTools:         []string{"cncf-exceptions-maker-0.1"},
		Created:              time.Now().Format("2006-01-02T15:04:05Z"),
	}

	return &spdx.Document2_1{
		CreationInfo: ci,
		Packages:     []*spdx.Package2_1{},
	}
}

// MakePackageFromRow creates an SPDX Package2_1 entry based on
// the contents of the spreadsheet row. It modifies and cleans up
// the data before returning the row.
func MakePackageFromRow(row []interface{}, rowNum int) (*spdx.Package2_1, error) {
	rd, err := convertRow(row)
	if err != nil {
		return nil, fmt.Errorf("unable to extract details from row: %v", err)
	}
	parseRowDetails(rd)
	cmt, err := prepComment(rd)

	pkg := &spdx.Package2_1{
		PackageName:             rd.componentName,
		PackageSPDXIdentifier:   fmt.Sprintf("SPDXRef-Package%d", rowNum),
		PackageDownloadLocation: "NOASSERTION",
		FilesAnalyzed:           false,
		PackageLicenseConcluded: rd.SPDXlicenses,
		PackageLicenseDeclared:  "NOASSERTION",
		PackageCopyrightText:    "NOASSERTION",
	}

	if rd.isComponentNameURL {
		pkg.PackageDownloadLocation = rd.componentName
	}

	if cmt != "" {
		pkg.PackageComment = cmt
	}

	return pkg, nil
}

type rowDetails struct {
	// extracted directly from row
	componentName         string
	githubRepo            string
	comments              string
	licenses              string
	SPDXlicenses          string
	approved              string
	whitelisted           string
	approvalMechanism     string
	notWhitelistedBecause string

	// filled in via parsing
	isComponentNameURL bool
}

func convertRow(row []interface{}) (*rowDetails, error) {
	// check that the row is the expected length
	if len(row) != 9 {
		return nil, fmt.Errorf("expected row of length %d, got %d", 9, len(row))
	}

	rd := rowDetails{}
	var ok bool

	rd.componentName, ok = row[0].(string)
	if !ok {
		return nil, fmt.Errorf("row[0] failed string type assertion, value is %v", row[0])
	}

	rd.githubRepo, ok = row[1].(string)
	if !ok {
		return nil, fmt.Errorf("row[1] failed string type assertion, value is %v", row[1])
	}

	rd.comments, ok = row[2].(string)
	if !ok {
		return nil, fmt.Errorf("row[2] failed string type assertion, value is %v", row[2])
	}

	rd.licenses, ok = row[3].(string)
	if !ok {
		return nil, fmt.Errorf("row[3] failed string type assertion, value is %v", row[3])
	}

	rd.SPDXlicenses, ok = row[4].(string)
	if !ok {
		return nil, fmt.Errorf("row[4] failed string type assertion, value is %v", row[4])
	}

	rd.approved, ok = row[5].(string)
	if !ok {
		return nil, fmt.Errorf("row[5] failed string type assertion, value is %v", row[5])
	}

	rd.whitelisted, ok = row[6].(string)
	if !ok {
		return nil, fmt.Errorf("row[6] failed string type assertion, value is %v", row[6])
	}

	rd.approvalMechanism, ok = row[7].(string)
	if !ok {
		return nil, fmt.Errorf("row[7] failed string type assertion, value is %v", row[7])
	}

	rd.notWhitelistedBecause, ok = row[8].(string)
	if !ok {
		return nil, fmt.Errorf("row[8] failed string type assertion, value is %v", row[8])
	}

	return &rd, nil
}

func parseRowDetails(rd *rowDetails) {
	_, err := url.ParseRequestURI(rd.componentName)
	rd.isComponentNameURL = (err == nil)
}

func prepComment(rd *rowDetails) (string, error) {
	cmts := []string{}
	if rd.comments != "" {
		cmts = append(cmts, rd.comments)
	}
	if rd.whitelisted == "Yes" {
		cmts = append(cmts, "whitelisted")
	} else if rd.whitelisted == "N/A" {
		if rd.approvalMechanism == "Apache-2.0 license" {
			cmts = append(cmts, "Apache-2.0, no approval needed")
		} else {
			return "", fmt.Errorf("N/A for whitelisted but not Apache-2.0: %v", rd)
		}
	} else {
		cmts = append(cmts, fmt.Sprintf("not whitelisted because: %s; approved by %s", rd.notWhitelistedBecause, rd.approvalMechanism))
	}

	return strings.Join(cmts, "; "), nil
}
