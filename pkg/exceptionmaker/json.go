// SPDX-License-Identifier: Apache-2.0
// Copyright The Linux Foundation

package exceptionmaker

import "github.com/spdx/tools-golang/v0/spdx"

// PackageSubset includes just a small subset of Package fields
// for JSON output.
type PackageSubset struct {
	Pkg     string `json:"package"`
	License string `json:"license"`
	Comment string `json:"comment"`
}

// ConvertSPDXToJSONPackageSubset does what it says on the tin.
func ConvertSPDXToJSONPackageSubset(doc *spdx.Document2_1) []PackageSubset {
	subsets := []PackageSubset{}

	for _, pkg := range doc.Packages {
		ps := PackageSubset{
			Pkg:     pkg.PackageName,
			License: pkg.PackageLicenseConcluded,
			Comment: pkg.PackageComment,
		}

		subsets = append(subsets, ps)
	}

	return subsets
}
