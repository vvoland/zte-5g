package zte

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"

	"github.com/rs/zerolog/log"
	"grono.dev/zte-5g/pkg/zteapi"
)

// ErrSessionExpired is returned when the session is expired.
var ErrSessionExpired = errors.New("session expired")

type Session struct {
	fwVersion string
	api       zteapi.HttpWithSession
}

// Connect connects to the modem and logs in.
// url is the modem's URL
// passSha256 is the SHA256 digest of the password.
func Connect(url url.URL, passSha256 string) (*Session, error) {
	if len(passSha256) != 64 {
		panic("passSha256 doesn't seem to be a valid SHA256")
	}
	api := zteapi.Http{
		Url:        url,
		PassSha256: passSha256,
	}

	ld, err := api.GetLD()
	log.Debug().Err(err).Str("ld", ld).Msg("get ld")
	if err != nil {
		return nil, fmt.Errorf("failed to get login parameters: %w", err)
	}

	session, err := api.Login(ld)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	apiWithSess := zteapi.HttpWithSession{
		Http:    api,
		Session: session,
	}

	var ver struct {
		Version string `json:"wa_inner_version"`
	}
	err = apiWithSess.GetCmd(&ver, "wa_inner_version")
	log.Debug().Err(err).Str("fwVersion", ver.Version).Msg("get wa_inner_version")
	if err != nil {
		return nil, fmt.Errorf("failed to get fw version: %w", err)
	}

	log.Info().Msg("connected successfully")
	return &Session{
		api:       apiWithSess,
		fwVersion: ver.Version,
	}, nil
}

// GetCmd sends a command to the modem and decodes the JSON response into out.
func (s *Session) GetCmd(out interface{}, cmd ...string) error {
	if len(cmd) == 0 {
		cmd = getJsonTags(out)
	}
	var all map[string]interface{}

	err := s.api.GetCmd(&all, append(cmd, "loginfo")...)

	if all["loginfo"] != "ok" {
		return ErrSessionExpired
	}

	b, _ := json.Marshal(all)
	json.Unmarshal(b, out)
	return err
}

// Close logs out from the modem.
func (s *Session) Close() error {
	rd, err := s.api.GetRD()
	log.Debug().Err(err).Str("rd", rd).Msg("get rd")
	if err != nil {
		return fmt.Errorf("failed to get logout parameter: %w", err)
	}

	// Session ended
	if !s.api.Alive() {
		return nil
	}

	ad := zteapi.EncodeAD(s.fwVersion, rd)
	log.Debug().Str("ad", ad).Str("fwVersion", s.fwVersion).Msg("encode ad")

	err = s.api.Logout(ad)
	if err != nil {
		log.Err(err).Msg("logout failed")
		return err
	} else {
		log.Debug().Err(err).Msg("logout")
	}

	log.Info().Msg("disconnected successfully")
	return nil
}

// get `json:"XXX,omitempty"` tags from struct fields
func getJsonTags(m any) []string {
	if reflect.TypeOf(m).Kind() == reflect.Ptr {
		m = reflect.ValueOf(m).Elem().Interface()
	}

	t := reflect.TypeOf(m)
	if t.Kind() != reflect.Struct {
		panic("expected struct")
	}

	var tags []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tags = append(tags, f.Tag.Get("json"))
	}
	return tags
}
