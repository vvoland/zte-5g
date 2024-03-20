package zteapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
)

type Http struct {
	Url        url.URL
	PassSha256 string
}

func (z *Http) GetLD() (string, error) {
	url := z.Url
	url.Path = "/goform/goform_get_cmd_process"

	var body struct {
		LD string `json:"LD"`
	}
	postBody := strings.NewReader(`isTest=false&cmd=LD`)
	req, err := http.NewRequest("POST", url.String(), postBody)
	if err != nil {
		return "", err
	}
	if _, err := httpDoRetry(req, 3, &body); err != nil {
		return "", err
	}

	return body.LD, nil
}

// Login logs in to the modem.
// Returns session token and error.
func (z *Http) Login(ld string) (string, error) {
	url := z.Url
	url.Path = "/goform/goform_set_cmd_process"

	var body struct {
		Result string `json:"result"`
	}

	encodedPass := EncodePass(z.PassSha256, ld)
	postBody := strings.NewReader("isTest=false&goformId=LOGIN&password=" + encodedPass)

	req, err := http.NewRequest("POST", url.String(), postBody)
	if err != nil {
		return "", err
	}
	resp, err := httpDoRetry(req, 3, &body)
	if err != nil {
		return "", err
	}

	if body.Result != "0" {
		log.Error().Str("status", body.Result).Msg("login failed")
		return "", nil
	}

	for _, c := range resp.Cookies() {
		if c.Name == "stok" {
			return c.Value, nil
		}
	}

	return "", fmt.Errorf("stok cookie not found")
}
