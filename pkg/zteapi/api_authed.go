package zteapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type HttpWithSession struct {
	Http
	Session string
}

func (z *HttpWithSession) Alive() bool {
	var body struct {
		LogInfo string `json:"loginfo"`
	}

	err := z.GetCmd(&body, "loginfo")
	if err != nil {
		return false
	}

	return body.LogInfo == "ok"
}

func (z *HttpWithSession) GetRD() (string, error) {
	var body struct {
		RD string `json:"RD"`
	}

	err := z.GetCmd(&body, "RD")
	if err != nil {
		return "", err
	}

	return body.RD, nil
}

func (z *HttpWithSession) GetCmd(out interface{}, cmds ...string) error {
	u := z.Url
	u.Path = "/goform/goform_get_cmd_process"

	q := url.Values{}
	q.Add("isTest", "false")
	q.Add("multi_data", "1")
	q.Add("cmd", strings.Join(cmds, ","))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}

	_, err = z.httpDo(req, out)
	if err != nil {
		return err
	}

	return nil
}

func (z *HttpWithSession) Logout(ad string) error {
	url := z.Url
	url.Path = "/goform/goform_set_cmd_process"

	var body struct {
		Result string `json:"result"`
	}

	postBody := strings.NewReader("isTest=false&goformId=LOGOUT&AD=" + ad)

	req, err := http.NewRequest("POST", url.String(), postBody)
	if err != nil {
		return err
	}
	_, err = z.httpDo(req, &body)
	if err != nil {
		return err
	}

	if body.Result != "success" {
		return fmt.Errorf("logout failed: %s", body.Result)
	}

	return nil
}

func (z *HttpWithSession) httpDo(req *http.Request, out interface{}) (*http.Response, error) {
	req.AddCookie(&http.Cookie{Name: "stok", Value: z.Session})
	return httpDoRetry(req, 3, out)
}
