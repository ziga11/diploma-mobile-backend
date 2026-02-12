package main

import (
	"backend/handlers"
	"backend/handlers/slack"
	"backend/types"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/robfig/cron/v3"
	"github.com/rs/cors"
)

var GoogleDrive *handlers.GoogleDriveService

func main() {
	handlers.MobileDB = startDB()
	gd, err := handlers.StartGoogleDrive()
	if err != nil {
		log.Printf("Failed to start google drive :%v", err)
		return
	}
	handlers.GoogleDrive = gd

	handler := Routing()
	CronJobs()

	log.Fatal(http.ListenAndServe((":8000"), *handler))

}

func startDB() *sql.DB {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is empty")
	}

	var err error
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("Failed open db: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed DB Ping: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(0)

	return db
}

func Routing() *http.Handler {
	router := mux.NewRouter()

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"POST", "GET", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	})

	handler := c.Handler(router)

	router.HandleFunc("/", handlers.WriteOK).Methods("GET")
	router.HandleFunc("/health", handlers.WriteOK).Methods("GET")
	router.HandleFunc("/googleAuth", handlers.SaveGoogleToken).Methods("GET")
	router.HandleFunc("/loginGoogle", handlers.StartGoogleAuth).Methods("GET")

	/* Auth */
	router.HandleFunc("/setPW", handlers.SetPWReq).Methods("PATCH")
	router.HandleFunc("/login", handlers.VerifyLogin).Methods("POST")

	/* Account */
	router.HandleFunc("/setLangCode", handlers.SetLangCode).Methods("PATCH")
	router.HandleFunc("/createAccount", handlers.ProcessBoardRequest).Methods("POST")
	router.HandleFunc("/accountById", handlers.AccountByIdReq).Methods("GET")
	router.HandleFunc("/resetPW", handlers.ResetPW).Methods("POST")

	/* Hashed Token */
	router.HandleFunc("/matchingHashedToken", handlers.MatchingTokenReq).Methods("GET")
	router.HandleFunc("/deleteHashedTokens", handlers.DeleteHashedTokensReq).Methods("DELETE")
	router.HandleFunc("/resetPwToken", handlers.ResetPwTokenReq).Methods("POST")

	/* Encrypted Token */
	router.HandleFunc("/saveEncryptedToken", handlers.InsertEncryptedTokenReq).Methods("POST")
	router.HandleFunc("/deleteEncryptedTokens", handlers.DeleteEncryptedTokensReq).Methods("DELETE")

	/* Slack*/
	router.HandleFunc("/slackResponse", slack.SlackInteraction).Methods("POST")
	router.HandleFunc("/slackCommandResponse", slack.SlackCommandResponse).Methods("POST")
	router.HandleFunc("/sendSlackFile", slack.AttachFilesToSlack).Methods("POST")
	router.HandleFunc("/sendSlackMsg", slack.SendMessageReq).Methods("POST")
	router.HandleFunc("/slackSubscription", slack.SlackSubscription).Methods("POST")

	/* Document */
	router.HandleFunc("/saveFile", handlers.SaveFile).Methods("POST")
	router.HandleFunc("/deleteDocument", handlers.DeleteDocument).Methods("DELETE")
	router.HandleFunc("/fetchDocById", handlers.FetchDocById).Methods("GET")
	router.HandleFunc("/fetchFile", handlers.FetchFile).Methods("GET")
	router.HandleFunc("/fetchDocs", handlers.UserDocs).Methods("GET")

	/* Obligation */
	router.HandleFunc("/setUserObligation", handlers.SetObligationStatusReq).Methods("PATCH")
	router.HandleFunc("/removeDocFromObligation", handlers.RemoveUserObligationDocsReq).Methods("DELETE")
	router.HandleFunc("/assignObligations", handlers.AssignObligationsReq).Methods("POST")
	router.HandleFunc("/userObligations", handlers.UserObligationsReq).Methods("GET")
	router.HandleFunc("/fetchObligationById", handlers.FetchObligationByID).Methods("GET")

	/* Notification */
	router.HandleFunc("/setNotificationSuitable", handlers.SetSuitableNotification).Methods("PATCH")
	router.HandleFunc("/readNotification", handlers.SetReadNotification).Methods("PATCH")
	router.HandleFunc("/fetchNotifications", handlers.FetchNotificationsReq).Methods("GET")

	/* Job */
	router.HandleFunc("/jobInfo", handlers.UserJobDetailsReq).Methods("GET")

	/* User */
	router.HandleFunc("/updateUserGroup", handlers.UpdateUserGroup).Methods("PATCH")
	router.HandleFunc("/userByID", handlers.UserById).Methods("GET")
	router.HandleFunc("/fetchUsers", handlers.FetchUsers).Methods("GET")
	router.HandleFunc("/updateUser", handlers.UpdateUserReq).Methods("PATCH")

	/* Message */
	router.HandleFunc("/fetchMessages", handlers.FetchMessagesReq).Methods("GET")
	router.HandleFunc("/fetchMsgThread", handlers.MessageThreadReq).Methods("GET")
	router.HandleFunc("/setReadMessage", handlers.SetReadStatusReq).Methods("PATCH")
	router.HandleFunc("/sendMessage", handlers.SendMessage).Methods("POST")

	return &handler
}

func CronJobs() {
	scheduler := cron.New()

	var delExpiredTokens = func() {
		handlers.DeleteHashedTokens(types.HashedTokenLookup{})
	}

	scheduler.AddFunc("0 0 * * *", delExpiredTokens)
	scheduler.AddFunc("0 * * * *", handlers.ObligationReminders)

	scheduler.Start()

}
