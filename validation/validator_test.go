// Copyright 2018 Kora ID. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package validation_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/enigma-id/go/validation"
	"github.com/stretchr/testify/assert"
)

func TestValidator_Field(t *testing.T) {
	t.Parallel()

	v := validation.New()
	r := v.Field(nil, "-")
	assert.True(t, r.Valid)

	r = v.Field(nil, "")
	assert.True(t, r.Valid)

	r = v.Field(nil, "nonexistingtag:1")
	assert.True(t, r.Valid)

	r = v.Field("5398228707871528", "cc")
	assert.True(t, r.Valid)

	var tests = []struct {
		value    interface{}
		param    string
		expected bool
	}{
		{false, "required", true},
		{nil, "required", false},
		{0, "numeric", true},
		{"abcd", "numeric", false},
		{0, "required|numeric", false},
		{"abcd", "alpha", true},
		{"abcd123", "alpha", false},
		{"abcd", "alpha_num", true},
		{"abcd123", "alpha_num", true},
		{"abcd123!@#", "alpha_num", false},
		{"abcd123!@#", "alpha_num_space", false},
		{"abcd 123", "alpha_num_space", true},
		{"foo@bar.com", "email", true},
		{"invalid.com", "email", false},
		{"https://foobar.com", "url", true},
		{"xyz://foobar.com", "url", false},
		{"123:f00", "json", false},
		{"{\"Name\":\"Alice\",\"Body\":\"Hello\",\"Time\":1294706395881547000}", "json", true},
		{"abcdefg", "lte:7", true},
		{"abcdefghij", "lte:7", false},
		{"abcdef", "gte:7", false},
		{"abcdefghij", "gte:7", true},
		{"abcdefg", "lt:7", false},
		{"abcdefg", "gt:7", false},
		{uint(2), "gt:7", false},
		{uint8(2), "gt:7", false},
		{float64(2.5), "gt:1", true},
		{float64(2.5), "gt:1.2", true},
		{float64(2.5), "gte:1.2", true},
		{float64(2.5), "lt:1.2", false},
		{float64(2.5), "lte:1.2", false},
		{float64(2.5), "gt:test", false},
		{float64(2.5), "gte:test", false},
		{float64(2.5), "lt:test", true},
		{float64(2.5), "lte:test", true},
		{"abcdef", "range:7,10", false},
		{"abcdefg", "range:7.2,8.4", false},
		{"abcdefgh", "range:7.7,10", true},
		{float64(7.8), "range:7.7,7.9", true},
		{float64(7.5), "range:7.7,7.9", false},
		{"abcdef", "range:abc,abcdefg", true},
		{"abcdefghij", "range:7,15", true},
		{"abacada", "contains:ritir", false},
		{"abacada", "contains:a", true},
		{"123456789", "match:[0-9]+", true},
		{"abacada", "match:cab$", false},
		{"123456789", "same:123546789", false},
		{"abacada", "same:abacada", true},
		{"abcd", "in:abcd,cdba", true},
		{"abcd", "in:abcde,cdba", false},
		{"abcd", "not_in:abcd,cdba", false},
		{"abcd", "not_in:abcde,cdba", true},
		{"abcd", "alpha|in:abcde,cdba", false},
	}

	for _, test := range tests {
		r := v.Field(test.value, test.param)
		assert.Equal(t, test.expected, r.Valid)
	}
}

func TestValidator_Struct(t *testing.T) {
	t.Parallel()

	type Address struct {
		Street string `valid:"-"`
		Zip    string `json:"zip" valid:"required"`
	}

	type User struct {
		Name         string `valid:"required"`
		Email        string `valid:"required|email"`
		Password     string `valid:"required|gte:7"`
		Age          int    `valid:"required|lte:30"`
		Home         *Address
		Works        []Address `valid:"required"`
		RegisteredAt time.Time `valid:"required"`
	}

	type Account struct {
		Name  string    `valid:"required"`
		Works []Address `valid:"required"`
	}

	type Slices struct {
		Name  string     `valid:"required"`
		Works []*Address `valid:"required"`
	}

	now := time.Now()
	var tests = []struct {
		param    interface{}
		expected bool
	}{
		{User{"John", "john@yahoo.com", "123G#678", 20, &Address{"Street", "123456"}, []Address{{"Street", "123456"}, {"Street", "123456"}}, now}, true},
		{User{"John Doe", "john@yahoo.com", "123G#678", 20, &Address{"Street", "123456"}, []Address{{"Street", "123456"}, {"Street", "123456"}}, now}, true},
		{User{"John Doe Doel", "john@yahoo.com", "123G#678", 20, &Address{"Street", "123456"}, []Address{{"Street", "123456"}, {"Street", "123456"}}, now}, true},
		{&User{"John", "john@yahoo.com", "123G#678", 20, &Address{"Street", "123456"}, []Address{{"Street", "123456"}, {"Street", "123456"}}, now}, true},
		{&User{"John", "john@yahoo.com", "123G#678", 20, &Address{"Street", "123456"}, []Address{}, now}, false},
		{User{"John-Doe", "john@yahoo.com", "123G#678", 20, &Address{"Street", "123456"}, []Address{{"Street", "123456"}, {"Street", "123456"}}, now}, true},
		{User{"John", "john@yahoo.com", "", 0, &Address{"Street", "123456"}, []Address{{"Street", "123456"}, {"Street", "123456"}}, now}, false},
		{User{"John", "john!yahoo.com", "12345678", 20, &Address{"Street", ""}, []Address{{"Street", "ABC456D89"}, {"Street", "123456"}}, now}, false},
		{User{"John", "john@yahoo.com", "123G#678", 20, &Address{"Street", "123456"}, []Address{{"Street", ""}, {"Street", "123456"}}, now}, false},
		{User{"John", "", "12345", 0, &Address{"Street", "123456789"}, []Address{{"", "ABC456D89"}, {"Street", "123456"}}, now}, false},
		{User{"John", "", "12345", 0, &Address{"Street", "123456789"}, []Address{{"", "ABC456D89"}, {"Street", "123456"}}, now}, false},
		{nil, false},
		{User{"John", "john@yahoo.com", "123G#678", 0, &Address{"Street", "123456"}, []Address{}, now}, false},
		{&User{Name: "John", Email: "john@yahoo.com", Password: "123G#678", Age: 20, Home: &Address{"Street", "123456"}, Works: []Address{{"Street", "123456"}, {"Street", "123456"}}}, false},
		{"im not a struct", false},
		{Account{Name: "John"}, false},
		{Account{"John", []Address{{"Street", "123456"}, {"Street", "123456"}}}, true},
		{Account{"John", []Address{{"", "123456"}, {"", ""}}}, false},
		{Account{"John", []Address{{"", ""}, {"Street", ""}}}, false},
		{Slices{Name: "John"}, false},
		{Slices{"John", []*Address{{"Street", "123456"}, {"Street", "123456"}}}, true},
		{Slices{"John", []*Address{{"Street", "123456"}, {"Street", ""}}}, false},
		{Slices{"John", []*Address{{"Street", ""}, {"Street", ""}}}, false},
	}

	v := validation.New()
	for xx, test := range tests {
		x := v.Struct(test.param)
		assert.Equal(t, test.expected, x.Valid, fmt.Sprintf("%d", xx))
	}
}

type User struct {
	ID   int
	Name string `valid:"required|match:[0-9]+"`
	Age  int    `valid:"required|range:1,140"`
}

type AnonymouseUser struct {
	ID2   int
	Name2 string `valid:"required|match:^(test)?\\w*@(/test/);com$"`
	Age2  int    `valid:"required|range:1,140"`
}

type Account struct {
	Username       string `valid:"required|gte:1|alpha_space"`
	Password       string `valid:"required|gte:3"`
	User           User   `valid:"required"`
	Members        []User `valid:"required"`
	Email          string `valid:"email"`
	MemberCode     string `valid:"alpha_space"`
	AnonymouseUser `valid:"-"`
}

func (t *Account) Validate() *validation.Response {
	o := &validation.Response{Valid: true}
	if t.Username != "" && len(t.Username) < 5 {
		o.Valid = false
		o.Failure("username.invalid", "username is not valid")
	}

	return o
}

func (t *Account) Messages() map[string]string {
	return map[string]string{
		"user.name.required":  "required",
		"password.required":   "required",
		"password.gte":        "more length please",
		"members.*.age.range": "invalid",
	}
}

func TestValidator_Request(t *testing.T) {
	type Acc struct {
		Slices    []Account  `valid:"required"`
		SlicesPtr []*Account `valid:"required"`
		Request   *Account   `valid:"required"`
	}

	acc := Account{
		Username: "",
		Email:    "valid@email.com",
		Password: "validpassword",
		User:     User{1, "1", 50},
		Members: []User{
			{1, "2", 50},
		},
	}

	accs := new(Acc)
	accs.Slices = append(accs.Slices, acc, acc)
	accs.SlicesPtr = append(accs.SlicesPtr, &acc, &acc)
	accs.Request = &acc

	v := validation.New()
	o := v.Struct(accs)
	assert.Error(t, o)

	assert.NotEmpty(t, o.GetMessage("slices.0.username.required"), "seharusnya ada error username dari slice validate request function")
	assert.NotEmpty(t, o.GetMessage("slices_ptr.0.username.required"), "seharusnya ada error username dari slice validate request function")
	assert.NotEmpty(t, o.GetMessage("request.username.required"), "seharusnya ada error username dari validate request function")

	u := &Account{
		Username: "x",
		Email:    "notemail.com",
		Password: "ab",
		User:     User{},
		Members: []User{
			{1, "jhon", 170},
		},
	}

	r := v.Request(u)
	assert.False(t, r.Valid)

	uv := &Account{
		Username: "validusername",
		Email:    "valid@email.com",
		Password: "validpassword",
		User:     User{1, "1", 50},
		Members: []User{
			{1, "2", 50},
		},
	}

	rv := v.Request(uv)
	assert.True(t, rv.Valid)
}

func TestResponse(t *testing.T) {
	r := &validation.Response{Valid: true}
	r.Failure("test", "ok")

	assert.Equal(t, r.Valid, false, "should be invalid")
	assert.Equal(t, "ok", r.GetMessage("test"))
}

func TestValidationErrorMessages(t *testing.T) {
	t.Parallel()

	v := validation.New()

	// field errors
	of := v.Field(nil, "required|numeric")
	assert.Equal(t, "The %s field is required", of.GetMessage("required"))

	// struct errors
	u := Account{Username: "use", Email: "notemail.com", Password: "abc123_", User: User{}, Members: []User{{1, "jhon", 170}}}
	os := v.Struct(u)

	assert.False(t, os.Valid)
	assert.Equal(t, 5, len(os.GetMessages()))
	assert.NotNil(t, os.Error())

	var tests = []struct {
		actual   string
		expected string
	}{
		{os.GetMessage("email.email"), "The email must be a valid email address"},
		{os.GetMessage("user.name.required"), "The name field is required"},
		{os.GetMessage("user.age.required"), "The age field is required"},
		{os.GetMessage("members.0.age.range"), "The age must be between 1 and 140"},
		{os.GetMessage("members.0.name.match"), "The name format is invalid"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.actual)
	}

	// requests errors
	ore := v.Request(&u)
	assert.Equal(t, 6, len(ore.GetMessages()))

	tests = []struct {
		actual   string
		expected string
	}{
		{ore.GetMessage("user.name.required"), "required"},
		{ore.GetMessage("user.age.required"), "The age field is required"},
		{ore.GetMessage("members.0.age.range"), "invalid"},
		{ore.GetMessage("members.0.name.match"), "The name format is invalid"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.actual)
	}

	u = Account{Username: ""}
	ore = v.Request(&u)

	assert.Equal(t, "The username field is required", ore.GetMessage("username.required"))
}

func TestResponse_Error(t *testing.T) {
	v := validation.New()

	u := Account{Username: ""}
	re := v.Request(&u)

	jsonString, _ := json.Marshal(re.GetErrors())
	assert.Equal(t, `{"members":"The members field is required","password":"required","user.age":"The age field is required","user.name":"required","username":"The username field is required"}`, string(jsonString))
}

func TestSetError(t *testing.T) {
	e := validation.SetError("email", "email is not valid")
	assert.Equal(t, "email is not valid", e.GetMessage("email"))
}
