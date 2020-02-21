package linebot

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	cloudkms "cloud.google.com/go/kms/apiv1"
	"github.com/line/line-bot-sdk-go/linebot"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

var (
	lineInfo LineInfo
)

// LineInfo LineのMessagingAPIを使用するためのあれこれ
type LineInfo struct {
	LineSecret string `json:"line_secret"`
	LineToken  string `json:"line_token"`
}

func init() {
	encode, err := ioutil.ReadFile("secrets.json.enc")
	if err != nil {
		log.Fatal("failed loading lineInfo", err)
		return
	}
	ctx := context.Background()
	client, err := cloudkms.NewKeyManagementClient(ctx)
	if err != nil {
		log.Fatal("failed loading lineInfo", err)
		return
	}
	req := &kmspb.DecryptRequest{
		Name:       lineSecretsKmsKeyName(),
		Ciphertext: encode,
	}
	resp, err := client.Decrypt(ctx, req)
	if err != nil {
		log.Fatal("failed loading lineInfo", err)
		return
	}
	secretJSON := resp.Plaintext
	if err := json.Unmarshal(secretJSON, &lineInfo); err != nil {
		log.Fatal("failed json unmarshal secrets", err)
		return
	}
}
func lineSecretsKmsKeyName() string {
	prjID := os.Getenv("GCP_PROJECT_ID")
	keyRingName := os.Getenv("KMS_KEY_RING_NAME")
	keyName := os.Getenv("KMS_LINE_SECRETS_KEY_NAME")
	return fmt.Sprintf("projects/%s/locations/global/keyRings/%s/cryptoKeys/%s", prjID, keyRingName, keyName)
}

// Webhook Lineから呼び出される
func Webhook(w http.ResponseWriter, r *http.Request) {
	client, err := linebot.New(lineInfo.LineSecret, lineInfo.LineToken)
	if err != nil {
		http.Error(w, "Error: init line client", http.StatusBadRequest)
		log.Fatal(err)
		return
	}
	events, err := client.ParseRequest(r)
	if err != nil {
		http.Error(w, "Error: parse Request", http.StatusBadRequest)
		log.Fatal(err)
		return
	}
	for _, e := range events {
		switch e.Type {
		case linebot.EventTypeMessage:
			message := linebot.NewTextMessage("Test")
			_, err := client.ReplyMessage(e.ReplyToken, message).Do()
			if err != nil {
				log.Fatal("Error: messaging", err)
				continue
			}
		}
	}
	fmt.Println(w, "ok")
}
