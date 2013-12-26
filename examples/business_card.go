package main

import (
	. "github.com/abhiyerra/workmachine/app"
)

type BusinessCard struct {
	ImageUrl    InputField  `work_desc:"Use this information for the data below." work_id:"image_url" crowd_type:"image"`
	Name        OutputField `work_desc:"Find the name from the business card" work_id:"name"`
	Company     OutputField `work_desc:"Find the company from the business card" work_id:"company"`
	Email       OutputField `work_desc:"Find the email from the business card" work_id:"email"`
	PhoneNumber OutputField `work_desc:"Find the phone number from the business card" work_id:"phone_number"`
}

type BusinessCardVerify struct {
	ImageUrl InputField  `work_desc:"Use this information for the data below." work_id:"image_url" crowd_type:"image"`
	Name     InputField  `work_desc:"Find the name from the business card" work_id:"name"`
	Email    InputField  `work_desc:"Find the email from the business card" work_id:"email"`
	IsValid  OutputField `work_desc:"Is the information above valid?" work_id:"is_valid"`
}

func main() {
	business_cards := Task{
		Title:       "Business Card Fields",
		Description: "Enter the fields.",
		Price:       1,
		Tasks: []BusinessCard{
			BusinessCard{
				ImageUrl: "google.com",
			},
			// BusinessCard{
			// 	ImageUrl: "yahoo.com",
			// },
		},
	}

	serve := TermServe{}
	var backend Backender = serve
	NewBatch(business_cards).Run(backend)
}
