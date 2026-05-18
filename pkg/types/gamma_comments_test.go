package types

import (
	"encoding/json"
	"testing"
)

func TestCommentUnmarshalCurrentGammaProfileShape(t *testing.T) {
	var comment Comment
	err := json.Unmarshal([]byte(`{
		"id":"2933135",
		"body":"Why so many people moving from this market?",
		"parentEntityType":"Series",
		"parentEntityID":35,
		"userAddress":"0xf5aa8ba8f7f0ef81f7ff0365212e6550116b0376",
		"createdAt":"2026-05-18T06:47:21.255417Z",
		"updatedAt":"2026-05-18T06:47:29.475121Z",
		"profile":{
			"name":"Higuain76",
			"pseudonym":"Growing-Bidding",
			"baseAddress":"0xf5aa8ba8f7f0ef81f7ff0365212e6550116b0376",
			"profileImage":"https://example.com/profile.png"
		}
	}`), &comment)
	if err != nil {
		t.Fatalf("unmarshal comment: %v", err)
	}

	if comment.User.Address != "0xf5aa8ba8f7f0ef81f7ff0365212e6550116b0376" {
		t.Fatalf("address = %q", comment.User.Address)
	}
	if comment.User.Pseudonym != "Growing-Bidding" {
		t.Fatalf("pseudonym = %q", comment.User.Pseudonym)
	}
	if comment.User.ProfileImage != "https://example.com/profile.png" {
		t.Fatalf("profile image = %q", comment.User.ProfileImage)
	}
	if comment.ParentID == nil || *comment.ParentID != 35 {
		t.Fatalf("parent id = %v", comment.ParentID)
	}
}

func TestCommentUserUnmarshalLegacyUserShape(t *testing.T) {
	var comment Comment
	err := json.Unmarshal([]byte(`{
		"id":"c1",
		"body":"gm",
		"user":{
			"address":"0xabc",
			"pseudonym":"pix",
			"profileImage":""
		},
		"parentId":99
	}`), &comment)
	if err != nil {
		t.Fatalf("unmarshal comment: %v", err)
	}

	if comment.User.Address != "0xabc" || comment.User.Pseudonym != "pix" {
		t.Fatalf("legacy user was not preserved: %+v", comment.User)
	}
	if comment.ParentID == nil || *comment.ParentID != 99 {
		t.Fatalf("legacy parent id = %v", comment.ParentID)
	}
}
