/* Copyright (C) 2021 Mattias Ohlsson

This program is free software; you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation; either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software Foundation,
Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301  USA */

package sectoralarm

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
)

const (
	BaseURL = "https://mypagesapi.sectoralarm.net"
)

type Overview struct {
	Panel        Panel         `json:"Panel"`
	Locks        []interface{} `json:"Locks"`
	Smartplugs   []Smartplug   `json:"Smartplugs"`
	Temperatures []Temperature `json:"Temperatures"`
	Cameras      []interface{} `json:"Cameras"`
	Photos       []interface{} `json:"Photos"`
	Access       []string      `json:"Access"`
}

type Panel struct {
	PartialAvalible             bool `json:"PartialAvalible"`
	PanelQuickArm               bool `json:"PanelQuickArm"`
	PanelCodeLength             int  `json:"PanelCodeLength"`
	LockLanguage                int  `json:"LockLanguage"`
	SupportsApp                 bool `json:"SupportsApp"`
	SupportsInterviewServices   bool `json:"SupportsInterviewServices"`
	SupportsPanelUsers          bool `json:"SupportsPanelUsers"`
	SupportsTemporaryPanelUsers bool `json:"SupportsTemporaryPanelUsers"`
	SupportsRegisterDevices     bool `json:"SupportsRegisterDevices"`
	CanAddDoorLock              bool `json:"CanAddDoorLock"`
	CanAddSmartPlug             bool `json:"CanAddSmartPlug"`
	HasVideo                    bool `json:"HasVideo"`
	Wifi                        struct {
		WifiExist bool   `json:"WifiExist"`
		Serial    string `json:"Serial"`
	} `json:"Wifi"`
	PanelID             string      `json:"PanelId"`
	ArmedStatus         string      `json:"ArmedStatus"`
	PanelDisplayName    string      `json:"PanelDisplayName"`
	StatusAnnex         string      `json:"StatusAnnex"`
	PanelTime           string      `json:"PanelTime"`
	AnnexAvalible       bool        `json:"AnnexAvalible"`
	IVDisplayStatus     bool        `json:"IVDisplayStatus"`
	DisplayWizard       bool        `json:"DisplayWizard"`
	BookedStartDate     string      `json:"BookedStartDate"`
	BookedEndDate       string      `json:"BookedEndDate"`
	InstallationStatus  int         `json:"InstallationStatus"`
	InstallationAddress interface{} `json:"InstallationAddress"`
	WizardStep          int         `json:"WizardStep"`
	AccessGroup         int         `json:"AccessGroup"`
	SessionExpires      string      `json:"SessionExpires"`
	IsOnline            bool        `json:"IsOnline"`
}

type Smartplug struct {
	Consumption         interface{} `json:"Consumption"`
	DisplayScenarios    bool        `json:"DisplayScenarios"`
	ID                  string      `json:"Id"`
	Label               string      `json:"Label"`
	PanelID             interface{} `json:"PanelId"`
	SerialNo            string      `json:"SerialNo"`
	Scenarios           interface{} `json:"Scenarios"`
	Status              string      `json:"Status"`
	TimerActive         bool        `json:"TimerActive"`
	TimerEvents         interface{} `json:"TimerEvents"`
	TimerEventsSchedule interface{} `json:"TimerEventsSchedule"`
}

type Temperature struct {
	ID          interface{} `json:"Id"`
	Label       string      `json:"Label"`
	SerialNo    string      `json:"SerialNo"`
	Temperature string      `json:"Temprature"`
	DeviceID    interface{} `json:"DeviceId"`
}

type Client struct {
	userID     string
	password   string
	BaseURL    *url.URL
	version    string
	HTTPClient *http.Client
}

func NewClient(userID, password string) (*Client, error) {
	u, _ := url.Parse(BaseURL)

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &Client{
		userID:   userID,
		password: password,
		BaseURL:  u,
		HTTPClient: &http.Client{
			CheckRedirect: func(req *http.Request,
				via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Jar: jar,
		},
	}, nil
}

func getVersion() (string, error) {
	resp, err := http.Get(BaseURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	r := regexp.MustCompile(`"/Scripts/main.js\?(v[A-Z0-9_]*)"`)

	match := r.FindStringSubmatch(string(body))
	if len(match) < 2 {
		return "", errors.New("Can't find version string")
	}

	return match[1], nil
}

func (c *Client) Login() error {
	resp, err := c.HTTPClient.Post(c.BaseURL.String()+"/User/Login",
		"application/x-www-form-urlencoded",
		bytes.NewBufferString("userID="+c.userID+"&password="+c.password))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 302 {
		return errors.New("Can't login")
	}

	c.version, err = getVersion()
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetPanelList() (panels []Panel, err error) {
	resp, err := c.HTTPClient.Get(c.BaseURL.String() + "/Panel/GetPanelList/")
	if err != nil {
		return panels, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&panels)
	if err != nil {
		return nil, err
	}

	return panels, nil
}

func (c *Client) GetTemperatures(panelID string) (temperatures []Temperature, err error) {
	resp, err := c.HTTPClient.Post(c.BaseURL.String()+"/Panel/GetTempratures/",
		"application/json;charset=utf-8",
		bytes.NewBufferString(`{"id":"`+panelID+`","Version":"`+c.version+`"}`))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&temperatures)
	if err != nil {
		return nil, err
	}

	return temperatures, nil
}

func (c *Client) GetOverview(panelID string) (overview *Overview, err error) {
	resp, err := c.HTTPClient.Post(c.BaseURL.String()+"/Panel/GetOverview/",
		"application/json;charset=utf-8",
		bytes.NewBufferString(`{"id":"`+panelID+`","Version":"`+c.version+`"}`))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&overview)
	if err != nil {
		return nil, err
	}

	return overview, nil
}
