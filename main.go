package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"game-center-verifier/rabbitmq"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type GameCenterAuthData struct {
	PlayerID     string `json:"playerID"`
	BundleID     string `json:"bundleID"`
	PublicKeyURL string `json:"publicKeyUrl"`
	Signature    string `json:"signature"`
	Salt         string `json:"salt"`
	Timestamp    uint64 `json:"timestamp"`
}

func main() {
	goDotErr := godotenv.Load()
	if goDotErr != nil {
		log.Println("Error loading .env file")
	}

	gameCenterAuthQueue := rabbitmq.New("game_center_auth_queue", os.Getenv("FCMCG_RMQ_ADDR"), false, false, false)

	<-time.After(1 * time.Second)

	gameCenterAuthDone := make(chan struct{})
	go gameCenterAuthQueue.ConsumeRepeat(handleGameCenterAuth, gameCenterAuthDone)

	<-gameCenterAuthDone
}

func handleGameCenterAuth(body []byte, corrId string, replyTo string) bool {
	replyQueue := rabbitmq.New(replyTo, os.Getenv("FCMCG_RMQ_ADDR"), false, true, true)

	var authData GameCenterAuthData
	err := json.Unmarshal(body, &authData)
	if err != nil {
		_ = replyQueue.Push([]byte("false"), "", "")
	} else {
		playerId, err := verifyGameCenterAuth(&authData)
		if err != nil {
			_ = replyQueue.Push([]byte("false"), "", "")
		} else {
			playerInfo := PlayerInfo{PlayerId: playerId}
			playerInfoJson, err := json.Marshal(&playerInfo)
			if err != nil {
				return false
			}
			_ = replyQueue.Push(playerInfoJson, corrId, "")
		}
	}

	return true
}

func verifyGameCenterAuth(authData *GameCenterAuthData) (string, error) {
	// 공개 키 가져오기
	publicKey, err := fetchPublicKey(authData.PublicKeyURL)
	if err != nil {
		return "", errors.New("failed to fetch public key")
	}

	// 서명 검증
	isValid, err := verifySignature(authData, publicKey)
	if err != nil {
		return "", errors.New("failed to verify signature")
	}

	if !isValid {
		return "", errors.New("invalid signature")
	}

	// 타임스탬프 검증
	if !isTimestampValid(authData.Timestamp) {
		return "", errors.New("too old")
	}

	// 사용자 인증 성공 처리
	return authData.PlayerID, nil
}

func fetchPublicKey(publicKeyURL string) (*rsa.PublicKey, error) {
	// URL 검증
	if !isAppleURL(publicKeyURL) {
		return nil, errors.New("invalid publicKeyURL")
	}

	// HTTPS 요청을 통해 공개 키 데이터 가져오기
	resp, err := http.Get(publicKeyURL)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch public key")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 공개 키 파싱
	certificate, err := x509.ParseCertificate(body)
	if err != nil {
		return nil, err
	}

	rsaPublicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("invalid public key type")
	}

	return rsaPublicKey, nil
}

func isAppleURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := parsedURL.Hostname()
	// 애플의 도메인인지 확인
	if strings.HasSuffix(host, ".apple.com") {
		return true
	}

	return false
}

func verifySignature(authData *GameCenterAuthData, publicKey *rsa.PublicKey) (bool, error) {
	// 서명 및 솔트 디코딩
	signature, err := base64.StdEncoding.DecodeString(authData.Signature)
	if err != nil {
		return false, err
	}

	salt, err := base64.StdEncoding.DecodeString(authData.Salt)
	if err != nil {
		return false, err
	}

	// 빅 엔디안으로 타임스탬프 변환
	timestampBytes := make([]byte, 8)
	for i := uint(0); i < 8; i++ {
		timestampBytes[7-i] = byte(authData.Timestamp >> (i * 8))
	}

	// 메시지 생성
	var message []byte
	message = append(message, []byte(authData.PlayerID)...)
	message = append(message, []byte(authData.BundleID)...)
	message = append(message, timestampBytes...)
	message = append(message, salt...)

	// 해시 계산
	hashed := sha256.Sum256(message)

	// 서명 검증
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature)
	return err == nil, err
}

func isTimestampValid(timestamp uint64) bool {
	// 현재 시간과 비교하여 유효한 범위 내에 있는지 확인
	now := time.Now().UnixMilli()
	delta := time.Duration((int64(timestamp) - now) * int64(time.Millisecond))
	if delta > 60 * time.Second || delta < -60 * time.Second {
		return false
	}
	return true
}
