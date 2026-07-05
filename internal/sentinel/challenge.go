package sentinel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

func FetchSentinelChallenge(session tls_client.HttpClient, deviceID, flow, ua, secChUA, impersonate string) (map[string]any, error) {
	generator := NewGenerator(deviceID, ua)
	reqBody := map[string]any{
		"p":    generator.GenerateRequirementsToken(),
		"id":   deviceID,
		"flow": flow,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := fhttp.NewRequest("POST", "https://sentinel.openai.com/backend-api/sentinel/req", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	req.Header.Set("Referer", "https://sentinel.openai.com/sentinel/20260124ceb8/frame.html")
	req.Header.Set("Origin", "https://sentinel.openai.com")
	req.Header.Set("User-Agent", ua)
	if secChUA != "" {
		req.Header.Set("sec-ch-ua", secChUA)
	} else {
		req.Header.Set("sec-ch-ua", "\"Not:A-Brand\";v=\"99\", \"Google Chrome\";v=\"145\", \"Chromium\";v=\"145\"")
	}
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", "\"Windows\"")
	req.Header.Set("oai-device-id", deviceID)
	req.Header.Set("oai-language", "en-US")

	resp, err := session.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != fhttp.StatusOK {
		return nil, fmt.Errorf("sentinel challenge request failed with status: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func BuildSentinelToken(session tls_client.HttpClient, deviceID, flow, ua, secChUA, impersonate string) (string, error) {
	challenge, err := FetchSentinelChallenge(session, deviceID, flow, ua, secChUA, impersonate)
	if err != nil {
		return "", err
	}

	cValue, ok := challenge["token"].(string)
	if !ok || cValue == "" {
		return "", fmt.Errorf("invalid sentinel challenge token")
	}

	var pValue string
	generator := NewGenerator(deviceID, ua)

	powData, _ := challenge["proofofwork"].(map[string]any)
	required, _ := powData["required"].(bool)
	seed, _ := powData["seed"].(string)

	if required && seed != "" {
		difficulty, _ := powData["difficulty"].(string)
		pValue = generator.GenerateToken(seed, difficulty)
	} else {
		pValue = generator.GenerateRequirementsToken()
	}

	tokenData := map[string]string{
		"p":    pValue,
		"t":    "",
		"c":    cValue,
		"id":   deviceID,
		"flow": flow,
	}

	resultBytes, err := json.Marshal(tokenData)
	if err != nil {
		return "", err
	}

	return string(resultBytes), nil
}
