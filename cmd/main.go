package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/ncostamagna/axul_auth/auth"
	"github.com/ncostamagna/go-app-users-lab/internal/user"
	"github.com/ncostamagna/go-app-users-lab/pkg/bootstrap"
	"github.com/ncostamagna/go-app-users-lab/pkg/handler"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {

	_ = godotenv.Load()
	l := bootstrap.InitLogger()

	db, err := bootstrap.DBConnection()
	if err != nil {
		l.Fatal(err)
	}

	pagLimDef := os.Getenv("PAGINATOR_LIMIT_DEFAULT")
	if pagLimDef == "" {
		l.Fatal("paginator limit default is required")
	}

	ctx := context.Background()
	userRepo := user.NewRepo(l, db)
	a, err := auth.New("12312312")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	userSrv := user.NewService(l, a, userRepo)
	h := handler.NewUserHTTPServer(ctx, user.MakeEndpoints(userSrv, user.Config{LimPageDef: pagLimDef}))

	port := os.Getenv("PORT")
	address := fmt.Sprintf("127.0.0.1:%s", port)

	srv := &http.Server{
		Handler:      accessControl(h),
		Addr:         address,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	errCh := make(chan error)
	go func() {
		l.Println("listen in ", address)
		errCh <- srv.ListenAndServe()
	}()

	err = <-errCh
	if err != nil {
		log.Fatal(err)
	}
}

func accessControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS, HEAD, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept,Authorization,Cache-Control,Content-Type,DNT,If-Modified-Since,Keep-Alive,Origin,User-Agent,X-Requested-With")

		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}
