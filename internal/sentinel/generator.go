package sentinel

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type SentinelTokenGenerator struct {
	DeviceID         string
	UserAgent        string
	RequirementsSeed string
	SID              string
}

func NewGenerator(deviceID, ua string) *SentinelTokenGenerator {
	if deviceID == "" {
		deviceID = uuid.New().String()
	}
	if ua == "" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"
	}
	return &SentinelTokenGenerator{
		DeviceID:         deviceID,
		UserAgent:        ua,
		RequirementsSeed: fmt.Sprintf("%f", rand.Float64()),
		SID:              uuid.New().String(),
	}
}

func (g *SentinelTokenGenerator) getConfig() []any {
	now := time.Now().UTC()
	nowStr := now.Format("Mon Jan 02 2006 15:04:05 GMT+0000 (Coordinated Universal Time)")
	perfNow := rand.Float64()*49000 + 1000
	timeOrigin := float64(now.UnixNano()/1e6) - perfNow

	navProps := []string{
		"vendorSub", "productSub", "vendor", "maxTouchPoints",
		"scheduling", "userActivation", "doNotTrack", "geolocation",
		"connection", "plugins", "mimeTypes", "pdfViewerEnabled",
		"webkitTemporaryStorage", "webkitPersistentStorage",
		"hardwareConcurrency", "cookieEnabled", "credentials",
		"mediaDevices", "permissions", "locks", "ink",
	}
	navProp := navProps[rand.Intn(len(navProps))]
	navVal := fmt.Sprintf("%s-undefined", navProp)

	screenRes := "1920x1080"
	sdkJS := "https://sentinel.openai.com/sentinel/20260124ceb8/sdk.js"

	return []any{
		screenRes,
		nowStr,
		4294705152,
		0, // nonce placeholder
		g.UserAgent,
		sdkJS,
		nil,
		nil,
		"en-US",
		"en-US,en",
		rand.Float64(),
		navVal,
		[]string{"location", "implementation", "URL", "documentURI", "compatMode"}[rand.Intn(5)],
		[]string{"Object", "Function", "Array", "Number", "parseFloat", "undefined"}[rand.Intn(6)],
		perfNow,
		g.SID,
		"",
		[]int{4, 8, 12, 16}[rand.Intn(4)],
		timeOrigin,
	}
}

func (g *SentinelTokenGenerator) base64Encode(data any) string {
	raw, _ := json.Marshal(data)
	return base64.StdEncoding.EncodeToString(raw)
}

func (g *SentinelTokenGenerator) GenerateToken(seed string, difficulty string) string {
	if seed == "" {
		seed = g.RequirementsSeed
	}
	if difficulty == "" {
		difficulty = "0"
	}

	startTime := time.Now()
	config := g.getConfig()

	for i := 0; i < 500000; i++ {
		config[3] = i
		elapsed := time.Since(startTime).Milliseconds()
		config[9] = elapsed

		data := g.base64Encode(config)
		hashHex := FNV1a32(seed + data)

		if hashHex[:len(difficulty)] <= difficulty {
			return "gAAAAAB" + data + "~S"
		}
	}

	// Fallback error token (simplified)
	return "gAAAAAB" + "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D" + g.base64Encode("None")
}

func (g *SentinelTokenGenerator) GenerateRequirementsToken() string {
	config := g.getConfig()
	config[3] = 1
	config[9] = rand.Intn(45) + 5
	data := g.base64Encode(config)
	return "gAAAAAC" + data
}
