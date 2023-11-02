package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"sync"
	"time"
)

type Credential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type BaseClient struct {
	Credential
	httpClient *http.Client
	mtx        *sync.RWMutex

	connected    bool
	lastSyncedAt time.Time
	rMtx         *sync.RWMutex // another lock, but it is not strict so that we can leave early without waiting for the main lock
}

func NewBaseClientFromRawCookies(ctx context.Context, username string, rawCookies string) (*BaseClient, error) {
	header := http.Header{}
	header.Add("Cookie", rawCookies)
	req := http.Request{Header: header}
	cookies := req.Cookies()
	for _, c := range cookies {
		c.Path = "/"
		c.Domain = ".twitter.com"
	}

	return NewBaseClientFromCookies(ctx, username, cookies)
}

func NewBaseClientFromCookies(_ context.Context, username string, cookies []*http.Cookie) (*BaseClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	jar.SetCookies(twitterURL, cookies)

	return &BaseClient{
		Credential: Credential{
			Username: username,
			Password: "",
		},
		httpClient: &http.Client{Jar: jar},
		mtx:        &sync.RWMutex{},

		connected: true,
		rMtx:      &sync.RWMutex{},
	}, nil
}

func NewBaseClientFromPassword(ctx context.Context, username string, password string) (*BaseClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	baseClient := &BaseClient{
		Credential: Credential{
			Username: username,
			Password: password,
		},
		httpClient: &http.Client{Jar: jar},
		mtx:        &sync.RWMutex{},

		connected: false,
		rMtx:      &sync.RWMutex{},
	}

	err = baseClient.Login(ctx)
	if err != nil {
		return nil, err
	}

	return baseClient, nil
}

func (baseClient *BaseClient) Login(ctx context.Context) error {
	// this mutex ensure all pending
	baseClient.mtx.Lock()
	defer baseClient.mtx.Unlock()

	if baseClient.Password == "" {
		return fmt.Errorf("failed to login: missing credential")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}

	baseClient.rMtx.Lock()
	baseClient.connected = false
	baseClient.rMtx.Unlock()

	baseClient.httpClient = &http.Client{Jar: jar}

	err = baseClient.initGuessToken(ctx)
	if err != nil {
		err2 := baseClient.initGuessToken2(ctx)
		if err2 != nil {
			return fmt.Errorf("failed to init guess token: %v; %v", err, err2)
		}
	}

	flowToken, err := baseClient.startLoginFlow(ctx)
	if err != nil {
		return fmt.Errorf("failed to start login flow: %s", err)
	}

	flowToken, err = baseClient.loginJsInstrumentationSubtask(ctx, flowToken)
	if err != nil {
		return fmt.Errorf("LoginEnterUserIdentifierSSO failed: %s", err)
	}

	flowToken, err = baseClient.loginEnterUserIdentifierSSO(ctx, flowToken, baseClient.Username)
	if err != nil {
		return fmt.Errorf("LoginEnterUserIdentifierSSO failed: %s", err)
	}

	flowToken, err = baseClient.loginEnterPassword(ctx, flowToken, baseClient.Password)
	if err != nil {
		return fmt.Errorf("LoginEnterPassword failed: %s", err)
	}

	_, err = baseClient.accountDuplicationCheck(ctx, flowToken)
	if err != nil {
		return fmt.Errorf("AccountDuplicationCheck failed: %s", err)
	}

	err = baseClient.ensureSearchSafety(ctx)
	if err != nil {
		log.Printf("[WARN] (%s) cannot enable search safety, however it is enabled by default anyway: %s", baseClient.Username, err)
	}

	baseClient.rMtx.Lock()
	baseClient.connected = true
	baseClient.lastSyncedAt = time.Now()
	baseClient.rMtx.Unlock()

	return nil
}

func (baseClient *BaseClient) DoRequestWithAuth(req *http.Request) (*http.Response, error) {
	return baseClient.doRequestWithAuth(req, true)
}

func (baseClient *BaseClient) doRequestWithAuth(req *http.Request, requireLock bool) (*http.Response, error) {
	if requireLock {
		baseClient.mtx.RLock()
		defer baseClient.mtx.RUnlock()
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", defaultBearerToken))
	for _, c := range baseClient.httpClient.Jar.Cookies(req.URL) {
		if c.Name == "ct0" {
			req.Header.Set("X-Csrf-Token", c.Value)
			continue
		}
		if c.Name == "auth_token" {
			req.Header.Set("X-Twitter-Auth-Type", "OAuth2Session")
			continue
		}
	}
	return baseClient.httpClient.Do(req)
}

func (baseClient *BaseClient) CanReconnect() bool {
	return baseClient.Password != ""
}

func (baseClient *BaseClient) Connected() bool {
	baseClient.rMtx.RLock()
	defer baseClient.rMtx.RUnlock()

	return baseClient.connected
}

func (baseClient *BaseClient) Disconnect() {
	baseClient.rMtx.Lock()
	defer baseClient.rMtx.Unlock()

	baseClient.connected = false
}

func (baseClient *BaseClient) LastSyncedAt() time.Time {
	baseClient.rMtx.RLock()
	defer baseClient.rMtx.RUnlock()

	return baseClient.lastSyncedAt
}

func (baseClient *BaseClient) initGuessToken(ctx context.Context) error {
	req, err := http.NewRequest(http.MethodGet, "https://twitter.com/", nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	resp, err := baseClient.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code %d", resp.StatusCode)
	}

	bodyBz, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	matches := regexp.MustCompile(`gt=(\d+)`).FindSubmatch(bodyBz)
	if len(matches) != 2 {
		return fmt.Errorf("cannot extract guess token, len(matches) = %d", len(matches))
	}

	cookies := baseClient.httpClient.Jar.Cookies(req.URL)
	cookies = append(cookies, &http.Cookie{
		Name:   "gt",
		Value:  string(matches[1]),
		Path:   "/",
		Domain: ".twitter.com",
		MaxAge: 10800,
		Secure: true,
	})
	baseClient.httpClient.Jar.SetCookies(req.URL, cookies)

	return nil
}

func (baseClient *BaseClient) initGuessToken2(ctx context.Context) error {
	req, err := http.NewRequest(http.MethodPost, "https://api.twitter.com/1.1/guest/activate.json", nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", defaultBearerToken))

	resp, err := baseClient.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code %d", resp.StatusCode)
	}

	bodyBz, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var respBody struct {
		GuestToken string `json:"guest_token"`
	}
	err = json.Unmarshal(bodyBz, &respBody)
	if err != nil {
		return err
	}

	cookies := baseClient.httpClient.Jar.Cookies(req.URL)
	cookies = append(cookies, &http.Cookie{
		Name:   "gt",
		Value:  respBody.GuestToken,
		Path:   "/",
		Domain: ".twitter.com",
		MaxAge: 10800,
		Secure: true,
	})
	baseClient.httpClient.Jar.SetCookies(req.URL, cookies)

	return nil
}

func (baseClient *BaseClient) startLoginFlow(ctx context.Context) (string, error) {
	reqBodyBz := []byte(`{"input_flow_data":{"flow_context":{"debug_overrides":{},"start_location":{"location":"unknown"}}},"subtask_versions":{"action_list":2,"alert_dialog":1,"app_download_cta":1,"check_logged_in_account":1,"choice_selection":3,"contacts_live_sync_permission_prompt":0,"cta":7,"email_verification":2,"end_flow":1,"enter_date":1,"enter_email":2,"enter_password":5,"enter_phone":2,"enter_recaptcha":1,"enter_text":5,"enter_username":2,"generic_urt":3,"in_app_notification":1,"interest_picker":3,"js_instrumentation":1,"menu_dialog":1,"notifications_permission_prompt":2,"open_account":2,"open_home_timeline":1,"open_link":1,"phone_verification":4,"privacy_options":1,"security_key":3,"select_avatar":4,"select_banner":2,"settings_list":7,"show_code":1,"sign_up":2,"sign_up_review":4,"tweet_selection_urt":1,"update_users":1,"upload_media":1,"user_recommendations_list":4,"user_recommendations_urt":1,"wait_spinner":3,"web_modal":1}}`)
	req, err := http.NewRequest(http.MethodPost, "https://api.twitter.com/1.1/onboarding/task.json?flow_name=login", bytes.NewReader(reqBodyBz))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", defaultBearerToken))
	cookies := baseClient.httpClient.Jar.Cookies(req.URL)
	for _, c := range cookies {
		if c.Name == "gt" {
			req.Header.Set("X-Guest-Token", c.Value)
			break
		}
	}
	req = req.WithContext(ctx)

	resp, err := baseClient.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response code %d", resp.StatusCode)
	}

	bodyBz, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var respBody struct {
		FlowToken string `json:"flow_token"`
	}
	err = json.Unmarshal(bodyBz, &respBody)
	if err != nil {
		return "", err
	}

	return respBody.FlowToken, nil
}

func (baseClient *BaseClient) loginJsInstrumentationSubtask(ctx context.Context, flowToken string) (string, error) {
	reqBodyBz, _ := json.Marshal(map[string]interface{}{
		"flow_token": flowToken,
		"subtask_inputs": []map[string]interface{}{
			{
				"subtask_id": "LoginJsInstrumentationSubtask",
				"js_instrumentation": map[string]interface{}{
					"response": "{}",
					"link":     "next_link",
				},
			},
		},
	})
	req, err := http.NewRequest(http.MethodPost, "https://api.twitter.com/1.1/onboarding/task.json", bytes.NewReader(reqBodyBz))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	cookies := baseClient.httpClient.Jar.Cookies(req.URL)
	for _, c := range cookies {
		if c.Name == "gt" {
			req.Header.Set("X-Guest-Token", c.Value)
			break
		}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", defaultBearerToken))
	req = req.WithContext(ctx)

	resp, err := baseClient.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response code %d", resp.StatusCode)
	}

	bodyBz, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var respBody struct {
		FlowToken string `json:"flow_token"`
	}
	err = json.Unmarshal(bodyBz, &respBody)
	if err != nil {
		return "", err
	}

	return respBody.FlowToken, nil
}

func (baseClient *BaseClient) loginEnterUserIdentifierSSO(ctx context.Context, flowToken string, username string) (string, error) {
	reqBodyBz, _ := json.Marshal(map[string]interface{}{
		"flow_token": flowToken,
		"subtask_inputs": []map[string]interface{}{
			{
				"subtask_id": "LoginEnterUserIdentifierSSO",
				"settings_list": map[string]interface{}{
					"setting_responses": []map[string]interface{}{
						{
							"key": "user_identifier",
							"response_data": map[string]interface{}{
								"text_data": map[string]interface{}{
									"result": username,
								},
							},
						},
					},
					"link": "next_link",
				},
			},
		},
	})
	req, err := http.NewRequest(http.MethodPost, "https://api.twitter.com/1.1/onboarding/task.json", bytes.NewReader(reqBodyBz))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	cookies := baseClient.httpClient.Jar.Cookies(req.URL)
	for _, c := range cookies {
		if c.Name == "gt" {
			req.Header.Set("X-Guest-Token", c.Value)
			break
		}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", defaultBearerToken))
	req = req.WithContext(ctx)

	resp, err := baseClient.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response code %d", resp.StatusCode)
	}

	bodyBz, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var respBody struct {
		FlowToken string `json:"flow_token"`
	}
	err = json.Unmarshal(bodyBz, &respBody)
	if err != nil {
		return "", err
	}

	return respBody.FlowToken, nil
}

func (baseClient *BaseClient) loginEnterPassword(ctx context.Context, flowToken string, password string) (string, error) {
	reqBodyBz, _ := json.Marshal(map[string]interface{}{
		"flow_token": flowToken,
		"subtask_inputs": []map[string]interface{}{
			{
				"subtask_id": "LoginEnterPassword",
				"enter_password": map[string]interface{}{
					"password": password,
					"link":     "next_link",
				},
			},
		},
	})
	req, err := http.NewRequest(http.MethodPost, "https://api.twitter.com/1.1/onboarding/task.json", bytes.NewReader(reqBodyBz))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	cookies := baseClient.httpClient.Jar.Cookies(req.URL)
	for _, c := range cookies {
		if c.Name == "gt" {
			req.Header.Set("X-Guest-Token", c.Value)
			break
		}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", defaultBearerToken))
	req = req.WithContext(ctx)

	resp, err := baseClient.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response code %d", resp.StatusCode)
	}

	bodyBz, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var respBody struct {
		FlowToken string `json:"flow_token"`
	}
	err = json.Unmarshal(bodyBz, &respBody)
	if err != nil {
		return "", err
	}

	return respBody.FlowToken, nil
}

func (baseClient *BaseClient) accountDuplicationCheck(ctx context.Context, flowToken string) (string, error) {
	reqBodyBz, _ := json.Marshal(map[string]interface{}{
		"flow_token": flowToken,
		"subtask_inputs": []map[string]interface{}{
			{
				"subtask_id": "AccountDuplicationCheck",
				"check_logged_in_account": map[string]interface{}{
					"link": "AccountDuplicationCheck_false",
				},
			},
		},
	})
	req, err := http.NewRequest(http.MethodPost, "https://api.twitter.com/1.1/onboarding/task.json", bytes.NewReader(reqBodyBz))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	cookies := baseClient.httpClient.Jar.Cookies(req.URL)
	for _, c := range cookies {
		if c.Name == "gt" {
			req.Header.Set("X-Guest-Token", c.Value)
			break
		}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", defaultBearerToken))
	req = req.WithContext(ctx)

	resp, err := baseClient.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response code %d", resp.StatusCode)
	}

	bodyBz, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var respBody struct {
		FlowToken string `json:"flow_token"`
	}
	err = json.Unmarshal(bodyBz, &respBody)
	if err != nil {
		return "", err
	}

	return respBody.FlowToken, nil
}

func (baseClient *BaseClient) ensureSearchSafety(ctx context.Context) error {
	var twid string
	cookies := baseClient.httpClient.Jar.Cookies(twitterURL)
	for _, c := range cookies {
		if c.Name == "twid" {
			matches := regexp.MustCompile(`u=(\d+)`).FindStringSubmatch(c.Value)
			if len(matches) == 2 {
				twid = matches[1]
			}
		}
	}
	if twid == "" {
		return fmt.Errorf("failed to extract twid from cookies")
	}

	reqBodyBz, _ := json.Marshal(map[string]interface{}{
		"optInFiltering": true,
		"optInBlocking":  true,
	})
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("https://twitter.com/i/api/1.1/strato/column/User/%s/search/searchSafety", twid),
		bytes.NewReader(reqBodyBz),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	for _, c := range cookies {
		if c.Name == "gt" {
			req.Header.Set("X-Guest-Token", c.Value)
			break
		}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", defaultBearerToken))
	req = req.WithContext(ctx)

	resp, err := baseClient.doRequestWithAuth(req, false)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code %d", resp.StatusCode)
	}

	return nil
}
