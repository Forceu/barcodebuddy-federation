package helper

import (
	cryptorand "crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func GetIpAddress(r *http.Request) string {
	// Get IP from X-FORWARDED-FOR header
	ips := r.Header.Get("X-FORWARDED-FOR")
	splitIps := strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP := net.ParseIP(ip)
		if netIP != nil {
			return ip
		}
	}

	// Get IP from the X-REAL-IP header
	ip := r.Header.Get("X-REAL-IP")
	netIP := net.ParseIP(ip)
	if netIP != nil {
		return ip
	}

	// Get IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "undefined-ip"
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		return ip
	}
	return "undefined-ip"
}

func GetSecondsToMidnight() string {
	t := time.Now()
	return strconv.Itoa((24 * 60 * 60) - (60*60*t.Hour() + 60*t.Minute() + t.Second()))
}

func ByteCountSI(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

// A rune array to be used for pseudo-random string generation
var characters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// Used if unable to generate secure random string. A warning will be output
// to the CLI window
func generateUnsafeId(length int) string {
	log.Println("Warning! Cannot generate securely random ID!")
	b := make([]rune, length)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}

// Returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := cryptorand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded securely generated random string.
func GenerateRandomString(length int) string {
	b, err := generateRandomBytes(length)
	if err != nil {
		return generateUnsafeId(length)
	}
	result := cleanRandomString(base64.URLEncoding.EncodeToString(b))
	return result[:length]
}

// Removes special characters from string
func cleanRandomString(input string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Panicln(err)
	}
	return reg.ReplaceAllString(input, "")
}

func SecureStringEqual(str1, str2 string) bool {
	return subtle.ConstantTimeCompare([]byte(str1), []byte(str2)) == 1
}
