// © 2025 Vlad-Stefan Harbuz <vlad@vlad.website>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"regexp"
	"strconv"
)

var BANK_REFERENCE_REGEXP *regexp.Regexp = regexp.MustCompile("ose-[0-9a-f]{6}")

const TPL_BANK_DETAILS = `
<p>
	Please send a bank transfer using the following details. Make sure you
	include the reference, so we can identify your payment.
</p>

<dl>
	<dt>Beneficiary name</dt>
	<dd>Open Source Endowment Foundation</dd>
	<dt>Beneficiary address</dt>
	<dd>1209 Orange Street, Wilmington, DE 19801, USA</dd>
	<dt>Bank name</dt>
	<dd>Choice Financial Group</dd>
	<dt>Bank address</dt>
	<dd>4501 23rd Avenue S, Fargo, ND 58104, USA</dd>
	<dt>ABA routing number</dt>
	<dd>091311229</dd>
	<dt>IBAN/Account number</dt>
	<dd>202580213394</dd>
	<dt>SWIFT/BIC code</dt>
	<dd>CHFGUS44021</dd>
	<dt>Type of account</dt>
	<dd>Checking</dd>
	<dt>Amount</dt>
	<dd>{{ .UsdCentAmountString }} USD</dd>
	<dt>Reference</dt>
	<dd class="bank-reference">{{ .BankReference }}</dd>
</dl>
`

type BankDetails struct {
	BankInstructions string
	BankReference    string
}

type BankInstructionsParams struct {
	UsdCentAmountString string
	BankReference       string
}

func createBankDetails(
	donorEmail string,
	usdCentAmount int64,
) (BankDetails, error) {
	var buf bytes.Buffer
	usdCentAmountString := strconv.FormatFloat(
		float64(usdCentAmount)/100, 'f', -1, 64)

	h := md5.New()
	h.Write([]byte(donorEmail))
	h.Write([]byte(usdCentAmountString))
	bankReferenceHex := hex.EncodeToString(h.Sum(nil))[:6]
	bankReference := fmt.Sprintf("ose-%s", bankReferenceHex)

	t, err := template.New("bank-details").Parse(TPL_BANK_DETAILS)
	if err != nil {
		return BankDetails{}, err
	}

	t.Execute(&buf, BankInstructionsParams{
		UsdCentAmountString: usdCentAmountString,
		BankReference:       bankReference,
	})
	return BankDetails{
		BankInstructions: buf.String(),
		BankReference:    bankReference,
	}, nil
}
