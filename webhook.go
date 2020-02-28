package linebot

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	cloudkms "cloud.google.com/go/kms/apiv1"
	"github.com/Atsu-Imo/line-golang/model"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/line/line-bot-sdk-go/linebot"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

var (
	lineInfo LineInfo
	db       *gorm.DB
)

// LineInfo LineのMessagingAPIを使用するためのあれこれ
type LineInfo struct {
	LineSecret string `json:"line_secret"`
	LineToken  string `json:"line_token"`
}

func init() {
	// init line setting
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

	// init DB connection
	db, err = gorm.Open("postgres", dbInfo())
	if err != nil {
		log.Fatal("failed connecting database", err)
		return
	}
}

func dbInfo() string {
	connectionName := os.Getenv("POSTGRES_INSTANCE_CONNECTION_NAME")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DBNAME")
	return fmt.Sprintf("user=%s password=%s host=/cloudsql/%s/ dbname=%s", dbUser, dbPassword, connectionName, dbName)
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
		var reply *linebot.TextMessage
		switch e.Type {
		case linebot.EventTypeMessage:
			switch message := e.Message.(type) {
			case *linebot.TextMessage:
				titleCnd := message.Text
				var videos []model.Video
				today := time.Now()
				todayTruncated := today.Truncate(time.Hour).Add(-time.Duration(today.Hour()) * time.Hour)
				yesterday := today.Add(-time.Duration(24) * time.Hour)
				yesterdayTruncated := yesterday.Truncate(time.Hour).Add(-time.Duration(yesterday.Hour()) * time.Hour)
				db.Where("title LIKE ?", "%"+titleCnd+"%").Where("published_at BETWEEN ? AND ?", yesterdayTruncated, todayTruncated).Find(&videos)
				if len(videos) == 0 {
					reply = linebot.NewTextMessage("良さげなの見つかりませんでした！")
					break
				}
				shuffleVideos(videos)
				video := videos[0]
				videoTitle := video.Title
				videoURL := video.URL
				replyText := fmt.Sprintf(`%s
				%s`, videoTitle, videoURL)
				reply = linebot.NewTextMessage(replyText)
			default:
				reply = linebot.NewTextMessage("テキストメッセージ以外は未対応です！ごめんなさい")
			}
		}
		_, err := client.ReplyMessage(e.ReplyToken, reply).Do()
		if err != nil {
			log.Fatal("Error: messaging", err)
			continue
		}
	}
	fmt.Println(w, "ok")
}
func shuffleVideos(a []model.Video) {
	rand.Seed(time.Now().UnixNano())
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}
