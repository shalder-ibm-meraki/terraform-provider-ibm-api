// @APIVersion 1.0.0
// @APITitle Swagger IBM Cloud Provider API
// @APIDescription Swagger IBM Cloud Provider API
// @Contact sakshiag@in.ibm.com

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fvbock/endless"
	"github.com/gorilla/mux"
	mgo "gopkg.in/mgo.v2"

	"github.com/IBM-Cloud/terraform-provider-ibm-api/utils"
)

var (
	staticContent = flag.String("staticPath", "./swagger/swagger-ui", "Path to folder with Swagger UI")
)

// IndexHandler ..
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	var isJSONRequest bool

	if acceptHeaders, ok := r.Header["Accept"]; ok {
		for _, acceptHeader := range acceptHeaders {
			if strings.Contains(acceptHeader, "json") {
				isJSONRequest = true
				break
			}
		}
	}

	if isJSONRequest {
		w.Write([]byte(resourceListingJson))
	} else {
		http.Redirect(w, r, "/swagger-ui/", http.StatusFound)
	}
}

// APIDescriptionHandler ..
func APIDescriptionHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := strings.Trim(r.RequestURI, "/")

	if apiKey == "v1" {
		if json, ok := apiDescriptionsJson[apiKey]; ok {
			w.Write([]byte(json))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	} else {
		if json, ok := apiDescriptionsJsonV2[apiKey]; ok {
			w.Write([]byte(json))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config := utils.GetConfiguration()

	//session, err := mgo.Dial(fmt.Sprintf("%s:%s@%s:%d", config.Mongo.UserName, config.Mongo.Password, config.Mongo.Host, config.Mongo.Port))
	session, err := mgo.Dial("localhost")
	if err != nil {
		log.Fatalln("Could not create mongo db session", err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	ensureIndex(session)

	r := mux.NewRouter()

	r.HandleFunc("/", IndexHandler)

	r.PathPrefix("/swagger-ui").Handler(http.StripPrefix("/swagger-ui", http.FileServer(http.Dir(*staticContent))))

	for apiKey := range apiDescriptionsJson {
		log.Println("API :", apiKey)
		r.HandleFunc("/"+apiKey, APIDescriptionHandler)
	}

	for apiKey := range apiDescriptionsJsonV2 {
		log.Println("API :", apiKey)
		r.HandleFunc("/"+apiKey, APIDescriptionHandler)
	}

	r.HandleFunc("/v1/configuration", utils.ConfHandler(session)).Methods("POST")

	r.HandleFunc("/v1/configuration/{repo_name}", utils.ConfDeleteHandler).Methods("DELETE")

	r.HandleFunc("/v1/configuration/{repo_name}/plan", utils.PlanHandler(session)).Methods("POST")

	r.HandleFunc("/v1/configuration/{repo_name}/show", utils.ShowHandler(session)).Methods("POST")

	r.HandleFunc("/v1/configuration/{repo_name}/apply", utils.ApplyHandler(session)).Methods("POST")

	r.HandleFunc("/v1/configuration/{repo_name}/destroy", utils.DestroyHandler(session)).Methods("POST")

	r.HandleFunc("/v1/configuration/{repo_name}/{action}/{actionID}/log", utils.LogHandler).Methods("GET")

	r.HandleFunc("/v1/configuration/{repo_name}/{action}/{actionID}/status", utils.StatusHandler(session)).Methods("GET")

	r.HandleFunc("/v1/configuration/{repo_name}/{action}/{log_file}", utils.ViewLogHandler)

	r.HandleFunc("/v1/configuration/{repo_name}/{action}", utils.GetActionDetailsHandler(session)).Methods("GET")

	r.HandleFunc("/v2/configuration", utils.ConfHandler(session)).Methods("POST")

	r.HandleFunc("/v2/configuration/{repo_name}/import", utils.TerraformerImportHandler(session)).Methods("GET")

	r.HandleFunc("/v2/configuration/{repo_name}/{action}/{actionID}/log", utils.LogHandler).Methods("GET")

	r.HandleFunc("/v2/configuration/{repo_name}/{action}/{actionID}/status", utils.StatusHandler(session)).Methods("GET")

	r.HandleFunc("/v2/configuration/{repo_name}/statefile", utils.TerraformerStateHandler(session)).Methods("GET")

	r.HandleFunc("/v2/configuration/{repo_name}/statefile", utils.TerraformerStateHandler(session)).Methods("POST")

	log.Println("Server will listen at port", config.Server.HTTPAddr, config.Server.HTTPPort)
	muxWithMiddlewares := http.TimeoutHandler(r, time.Second*60, "Timeout!")
	err = endless.ListenAndServe(fmt.Sprintf("%s:%d", config.Server.HTTPAddr, config.Server.HTTPPort), muxWithMiddlewares)
	if err != nil {
		log.Println("Couldn't start the server", err)
	}
}

func ensureIndex(s *mgo.Session) {
	session := s.Copy()
	defer session.Close()
	c := session.DB("action").C("actionDetails")

	index := mgo.Index{
		Key:        []string{"actionid"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	err := c.EnsureIndex(index)
	if err != nil {
		panic(err)
	}
}
