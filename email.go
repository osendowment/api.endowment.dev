// © 2025 Vlad-Stefan Harbuz <vlad@vlad.website>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"github.com/resend/resend-go/v2"
	"html/template"
	"log"
)

type NewDonationParams struct {
	Email              string
	Name               *string
	UsdCentAmountFloat float64
	DonationMethod     string
	BankReference      *string
}

const TPL_NEW_DONATION = `
<p>
	Someone just made a donation.
</p>

<ul>
	<li>Email: {{ .Email }}</li>
	<li>Name: {{ .Name }}</li>
	<li>Amount: {{ .UsdCentAmountFloat }} USD</li>
	<li>Donation Method: {{ .DonationMethod }}</li>
	<li>Bank Reference: {{ .BankReference }}</li>
</ul>
`

type UnexpectedDonationParams struct {
	UsdCentAmountFloat float64
	DonationMethod     string
	BankReference      *string
}

const TPL_UNEXPECTED_DONATION = `
<p>
	We have just received a donation in the Mercury bank account that has an
	unknown bank reference. This donation will not be added to the donation
	table. It might be a good idea to investigate the situation.
</p>

<ul>
	<li>Amount: {{ .UsdCentAmountFloat }} USD</li>
	<li>Donation Method: {{ .DonationMethod }}</li>
	<li>Bank Reference: {{ .BankReference }}</li>
</ul>
`

type DuplicateDonationParams struct {
	Email              string
	UsdCentAmountFloat float64
	DonationMethod     string
	BankReference      *string
}

const TPL_DUPLICATE_DONATION = `
<p>
	We have just received a donation that might be a duplicate. That is, we
	have received a donation in the Mercury bank account, but its bank
	reference is already present in our donation table. It might be a good idea
	to investigate the situation.
</p>

<ul>
	<li>Email: {{ .Email }}</li>
	<li>Amount: {{ .UsdCentAmountFloat }} USD</li>
	<li>Donation Method: {{ .DonationMethod }}</li>
	<li>Bank Reference: {{ .BankReference }}</li>
</ul>
`

func sendStaffEmail(subject string, body string) {
	ctx := context.TODO()
	client := resend.NewClient(RESEND_API_KEY)

	params := &resend.SendEmailRequest{
		From: "Open Source Endowment <bot@endowment.dev>",
		// To:          []string{"donors@endowment.dev", "vlad@endowment.dev"},
		To:      []string{"vlad@vlad.website"},
		Subject: subject,
		Html:    body,
	}

	_, err := client.Emails.SendWithContext(ctx, params)

	if err != nil {
		log.Printf("Could not send email: %v", err)
		return
	}

	log.Printf("Sent email: %+v", params)
}

func sendNewDonationEmail(donor Donor, donation Donation) {
	t, err := template.New("new-donation").Parse(TPL_NEW_DONATION)
	if err != nil {
		log.Printf("Could not send new donation email: %v", err)
		return
	}

	var buf bytes.Buffer
	t.Execute(&buf, NewDonationParams{
		Email:              donor.Email,
		Name:               donor.Name,
		UsdCentAmountFloat: float64(donation.UsdCentAmount) / 100,
		DonationMethod:     donation.DonationMethod,
		BankReference:      donation.BankReference,
	})

	sendStaffEmail("New donation", buf.String())
}

func sendUnexpectedDonationEmail(donation Donation) {
	t, err := template.New("unexpected-donation").Parse(TPL_UNEXPECTED_DONATION)
	if err != nil {
		log.Printf("Could not send unexpected donation email: %v", err)
		return
	}

	var buf bytes.Buffer
	t.Execute(&buf, UnexpectedDonationParams{
		UsdCentAmountFloat: float64(donation.UsdCentAmount) / 100,
		DonationMethod:     donation.DonationMethod,
		BankReference:      donation.BankReference,
	})

	sendStaffEmail("Unexpected donation", buf.String())
}

func sendDuplicateDonationEmail(donation Donation) {
	t, err := template.New("duplicate-donation").Parse(TPL_DUPLICATE_DONATION)
	if err != nil {
		log.Printf("Could not send duplicate donation email: %v", err)
		return
	}

	var buf bytes.Buffer
	params := NewDonationParams{
		UsdCentAmountFloat: float64(donation.UsdCentAmount) / 100,
		DonationMethod:     donation.DonationMethod,
		BankReference:      donation.BankReference,
	}
	if donation.Donor == nil {
		params.Email = "UNKNOWN"
	} else {
		params.Email = *donation.Donor
	}
	t.Execute(&buf, params)

	sendStaffEmail("Duplicate donation", buf.String())
}
